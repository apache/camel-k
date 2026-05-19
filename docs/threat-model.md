# Apache Camel K — Threat Model

> This document describes the *implicit contract* between Apache Camel K and
> the people and clusters that run it: what Camel K assumes, what it
> guarantees, what it leaves to the operator/deployer, and the misuses that
> are syntactically possible but outside the intended use. It is **not** an
> audit, a pentest report, a CVE list, or a build-hygiene checklist.

## 4.1 Header

| Field | Value |
| --- | --- |
| Project | Apache Camel K |
| Version / commit | `2.11.0-SNAPSHOT`, commit `31286c9cf` (`main`) |
| Date | 2026-05-18 |
| Author(s) | Threat-model producer (draft-first); ratified by maintainer |
| Status | **ACCEPTED — ratified by maintainer (2026-05-18)** |

**Version binding.** This model is versioned with the project. A report
against Camel K version *N* is triaged against the model as it stood at *N*,
not at HEAD. The model is meaningfully tied to the *operator* version and the
*IntegrationPlatform* configuration in effect on the target cluster.

**Reporting cross-reference.** Camel K has no separate `SECURITY.md` and no
Camel-K-specific security page; the ASF cross-foundation security index lists
only "Apache Camel", governed by the ASF process (security@apache.org)
*(documented — `security.apache.org/projects/`)*. The Apache Camel
**Security Model** (`camel.apache.org/manual/security-model.html`) is the
Camel PMC's umbrella triage reference; it explicitly scopes itself to
"Camel … embedded in someone else's application, **not a multi-tenant
managed service**" *(documented — Camel Security Model, "Trust model")*.
Camel K is precisely the layer that operates Camel applications on a
cluster, so **this document is the Camel-K sub-project expansion of that
umbrella model for the cluster/operator/CR boundary the umbrella excludes**;
it is a strict superset and does not contradict the umbrella model.
Findings that violate a §4.8 property should be reported through the ASF
channel. Findings that fall under §4.3, §4.7, or §4.9 will be closed citing
this document.

**Provenance legend.** *(documented)* = stated in Camel K or Apache Camel
docs (cite source). *(maintainer)* = stated by a maintainer in response to
this process. *(inferred)* = reasoned from code structure or absence of a
feature; has a matching open question in §4.14. No *(inferred)* claims
remain.

**Ratification.** The model was **ratified by a maintainer on
2026-05-18**; every claim carries *(documented)* or *(maintainer,
2026-05-18)*. The model **closes vulnerability reports across all
sections**; §4.14 retains the dated ratification record (chain of
authority) and the one non-blocking meta item.

**Purpose (one paragraph).** Apache Camel K is a Kubernetes-native
integration platform. A user submits an *Integration* (or *Pipe* /
*Kamelet*) custom resource containing Camel route logic; the Camel K
*operator* — a privileged in-cluster controller — reconciles it: it resolves
Maven dependencies, builds a container image (with Maven + Jib, or S2I on
OpenShift), publishes the image to a registry, and creates the Kubernetes
workload (Deployment, Knative Service, or CronJob) that runs the route. The
`kamel` CLI is a thin client over the Kubernetes API that assembles and
submits these CRs. Camel K is therefore a *build-and-run control plane*: by
design it turns declarative input into executing code and container images
inside a cluster.

---

## 4.2 Scope and intended use

- **Primary intended use** *(documented — README.adoc, architecture.adoc)*:
  in-cluster build and execution of Apache Camel integrations described in
  Camel DSL, on Kubernetes/OpenShift, driven by an operator that owns the
  Integration→IntegrationKit→Build→workload pipeline.
- **Deployment shape** *(documented — operator.adoc, multi.adoc)*: a
  long-running operator Deployment in the cluster, installed either
  *namespaced* (namespace-scoped RBAC) or *global/descoped* (cluster-scoped
  RBAC, watches all namespaces). Plus the `kamel` CLI run by users against
  the Kube API.
- **Caller roles.** Camel K is a control plane, not an in-process library;
  the "caller" splits into distinct roles, each modeled separately:

  | Role | Trust level | Notes |
  | --- | --- | --- |
  | **Platform admin / operator-deployer** | Fully trusted | Installs the operator, sets its RBAC, edits `IntegrationPlatform` (registry, base image, Maven settings, build strategy). Compromise = total. |
  | **CR author** (submits Integration / Pipe / Kamelet / IntegrationKit / Build / IntegrationProfile) | Trusted *to run code in the target namespace* | RBAC-gated by the Kube API. Once gated, this role is **by design able to execute arbitrary code/containers** via the operator. *(maintainer, 2026-05-18 — Q2)* |
  | **Cluster tenant without Camel K CR RBAC** | Untrusted | Should not be able to cause builds/workloads. In scope as an adversary (§4.7). |
  | **Network client of the running integration** | Untrusted | Talks to the deployed route's own endpoints. Governed by **Apache Camel core's** security model, not Camel K's (§4.3). |

- **Component-family table.**

  | Family | Representative entry point | Touches outside the process? | In this model? |
  | --- | --- | --- | --- |
  | Operator / controllers | `pkg/controller/*`, reconcile of `camel.apache.org` CRs | Kube API, registry, Maven repos, GitHub (Kamelet repos) | **In** |
  | Build subsystem | `pkg/builder/*`, Build CR, `kamel builder` | Executes Maven/Jib/S2I; fetches deps, git, base image | **In** |
  | `kamel` CLI | `pkg/cmd/*`, `cmd/kamel` | Kube API; resolves local/`http`/`github`/`gist` sources client-side | **In** |
  | Integration runtime pod | the built image running user routes | Wherever the route's components point | **Out** — governed by Apache Camel core's threat model (§4.3) |
  | Samples / tests / proposals / deprecated tasks | `examples/`, `e2e/`, `proposals/`, Buildah/Kaniko/Spectrum tasks | n/a | **Out** (§4.3) |

- **What Camel K is not** *(maintainer, 2026-05-18 — Q1)*: not a sandbox,
  not a multi-tenant isolation boundary, not a supply-chain verification
  tool, not itself an internet-facing service (the operator listens only on
  health/metrics ports; CRs arrive via the Kube API server).

---

## 4.3 Out of scope (explicit non-goals)

- **The deployed integration's own runtime behavior.** Once Camel K has
  built and deployed the route, the route's attack surface (HTTP header
  handling, deserialization, expression languages, component
  vulnerabilities — e.g. the Camel-core header-injection class,
  CVE-2025-27636) is **Apache Camel core's** threat model and the route
  author's responsibility, not Camel K's. Camel K's model ends at "image
  built and workload created." *(documented — Camel Security Model: route
  authors and deployment operators are "fully trusted" and "Code execution
  by a route author is by design and is not a vulnerability in the
  framework"; external message senders are the untrusted attacker;
  `camel.apache.org/manual/security-model.html`. Camel-K boundary
  application: maintainer, 2026-05-18 — Q3)*
- **An adversary who already holds RBAC to create/patch Camel K CRs.** By
  design that principal can run code in the target namespace; they are not
  a meaningful adversary at that namespace's trust level (§4.7).
- **A platform admin / anyone who can edit `IntegrationPlatform`,
  `IntegrationProfile`, the operator Deployment, or the operator's RBAC.**
  They steer base image, registry, Maven settings, and build strategy for
  every integration; compromise here is total and not modeled *(maintainer,
  2026-05-18 — Q4: a CR author is trusted only at their own namespace's
  level; platform admin is a distinct, fully-trusted role)*.
- **Supply-chain trust of artifacts the platform/admin points at** — Maven
  Central or a configured Maven repo/mirror, the container registry, the
  base image, remote (GitHub) Kamelet catalogs. Camel K fetches and trusts
  these; choosing trustworthy sources is wholly a downstream responsibility
  (§4.10) and a well-known attack class (§4.9), not a Camel K vulnerability
  *(maintainer, 2026-05-18 — Q5)*.
- **Shipped-but-unsupported code.** `examples/`, `e2e/` (including test
  fixtures that grant cross-namespace SA permissions), `proposals/`, sample
  CRs under `pkg/resources/config/samples/`, and **deprecated API surface
  still present in CRD schemas** (`BuildahTask`, `KanikoTask`,
  `SpectrumTask`; the `pod` trait; multi-operator/`OPERATOR_ID`; synthetic
  Integrations) are not covered by this model — they are sample/test or
  superseded code, threat-modeled separately if at all *(maintainer,
  2026-05-18 — Q6; deprecation documented in `build_types.go`, `pod.go`,
  `multi.adoc`)*.

---

## 4.4 Trust boundaries and data flow

The dominant trust boundary is **"can create/patch a `camel.apache.org` CR"
→ arbitrary code and container execution inside the cluster, performed by a
high-privilege operator**, with **no admission/validation/mutating webhook
in between** *(maintainer, 2026-05-18 — Q7: the absence of any webhook is
intentional; there is no `admissionregistration` code)*. Kubernetes RBAC on
`camel.apache.org` resources is the *only* gate.

Data flow and the trust transitions it crosses:

1. **User → Kube API (`kamel` or `kubectl`).** Authn/authz is entirely the
   user's kubeconfig + Kube RBAC. The `kamel` CLI performs no privilege
   check of its own *(maintainer, 2026-05-18 — Q8; `pkg/cmd/run.go`)*.
   `kamel` additionally resolves `http`/`https`/`github`/`gist` source URLs
   **client-side** and embeds their contents into the CR — a trust
   transition on the *user's* workstation, not the operator's.
2. **CR (Integration/Pipe/Kamelet/IntegrationKit/Build/IntegrationProfile)
   → operator.** This is the primary boundary. Everything inside a CR the
   operator can read is treated as already authorized by Kube RBAC and is
   turned into builds, images, and pods. The operator runs with far broader
   privilege than a typical CR author (§4.5a / §4.7).
3. **Operator → build.** With the default `Routine` strategy the Maven
   build of user-supplied dependencies / git sources / Maven settings runs
   **inside the operator pod itself**; with the `Pod` strategy it runs in a
   separate builder pod under the `camel-k-builder` ServiceAccount
   *(documented — builds.adoc default `routine`; in-pod execution + posture
   maintainer, 2026-05-18 — Q9)*.
4. **Operator/builder → external network**: container registry (push/pull),
   Maven repositories (dependency download + plugin execution), GitHub
   (remote Kamelet catalogs). Each is a supply-chain edge (§4.9, §4.10).
5. **Integration pod → external systems**: bounded only by the pod's
   ServiceAccount and any NetworkPolicy (none ships with Camel K).

**Reachability preconditions per component (the triager's first test):**

- A finding in the **operator/controllers** is in-model only if it is
  reachable by a principal *without* RBAC to create the CR that triggers it
  (otherwise the principal is trusted per §4.7), **or** if it lets a CR
  author act beyond their own namespace's trust level — which is **in scope
  and `VALID`** *(maintainer, 2026-05-18 — Q4)*.
- A finding in the **build subsystem** is in-model only if it is reachable
  from CR-author-controlled input *and* does not merely restate that
  untrusted Maven/git input is built (which is §4.9, by design).
- A finding in the **`kamel` CLI** is in-model only if it affects a user
  other than the one running the CLI (the CLI runs with the invoking
  user's own privileges).
- A finding in the **integration runtime pod** is out of model here
  (§4.3) — route it to Apache Camel core.

---

## 4.5 Assumptions about the environment

- **Kubernetes / OpenShift** is the host. Camel K assumes a working API
  server, RBAC enforced by that API server, and an etcd-backed control
  plane *(documented — operator.adoc)*.
- **The Kube API server is the authority for authn/authz.** Camel K does
  not re-implement or second-guess RBAC; "the API let this CR be created"
  is taken as "this principal is authorized for it" *(maintainer,
  2026-05-18 — Q8)*.
- **A container registry** the operator can push to and the cluster can
  pull from, and **Maven repositories** reachable from the operator/builder
  *(documented — registry.adoc, builds.adoc)*.
- **Concurrency.** Multiple operators must not contend the same CRs;
  isolation between operators relies on the (deprecated) operator-id
  annotation *(documented — multi.adoc)*.
- **What Camel K does *to* its host** (negative inventory; *(maintainer,
  2026-05-18 — Q10)*):
  - Creates/updates: Deployments, Knative Services, CronJobs, Jobs,
    Services, Ingresses, Gateway HTTPRoutes, ConfigMaps, PVCs, PDBs, Pods,
    and **ServiceAccounts, Roles, RoleBindings, and ClusterRoleBindings**;
    reads Secrets; can `create` on **`pods/exec`** *(documented —
    `pkg/resources/config/rbac/*`)*.
  - With `Routine` build strategy it `exec`s Maven inside the operator pod.
  - It does **not** ship NetworkPolicies, does **not** install a webhook,
    and does **not** itself expose a network service beyond
    health/monitoring ports *(maintainer, 2026-05-18 — Q10)*.

---

## 4.5a Build-time and configuration variants

"Camel K" is a *family* of postures. The knobs that move the security
envelope:

| Knob | Default | Effect on the model | Maintainer stance |
| --- | --- | --- | --- |
| `BUILD_STRATEGY` (`routine` \| `pod`) | **`routine`** *(documented — builds.adoc)* | `routine`: untrusted Maven deps / git sources / Maven settings are fetched and Maven (with arbitrary plugins) executes **in the operator pod**, sharing the operator's ServiceAccount and blast radius. `pod`: build runs in a separate builder pod under `camel-k-builder`. | **RATIFIED 2026-05-18 (Q9):** `routine` is supported **only for trusted CR authors**; for untrusted/multi-tenant use `pod` is **required**, so build-time impact on the operator pod under `routine` with untrusted authors is `OUT-OF-MODEL: non-default-build`. |
| Install scope (`namespaced` \| `descoped`/global) | install-method dependent | Global gets a ClusterRole and watches all namespaces; namespaced confines RBAC to one namespace. | **RATIFIED 2026-05-18 (Q4):** author trusted only at own-namespace level; cross-ns/cluster escalation via the operator is in scope & `VALID`. |
| `builder` trait `tasks` (requires `Pod` strategy) | unset | Lets a CR author run an **arbitrary image with an arbitrary command** in the builder pod. By design. | **RATIFIED 2026-05-18 (Q2):** by-design code-exec; gated only by CR-create RBAC. |
| `container.image` / sourceless / synthetic Integration | unset | Runs an arbitrary externally-built image as the workload with no build/verification. | **RATIFIED 2026-05-18 (Q2):** by-design; gated only by CR-create RBAC. |
| `registry.insecure`, Maven `InsecureSkipVerify` | false | Disables TLS verification to registry / Maven repos. | **RATIFIED 2026-05-18 (Q5):** non-default and **unsupported in production**; impact is `OUT-OF-MODEL: non-default-build`. |
| `CAMEL_K_SYNTHETIC_INTEGRATIONS` | `false` *(documented as deprecated)* | Auto-adopts arbitrary Deployments/CronJobs/KnativeServices as Integrations. | **RATIFIED 2026-05-18 (Q6):** deprecated, out of the security model. |
| `security-context.runAsNonRoot` | **`false`** *(documented — security-context.adoc)* | Integration pods run as root unless overridden (or unless OpenShift assigns a UID). Looks like a hardening default; is not one. | **RATIFIED 2026-05-18 (Q11):** deliberate compatibility default; enforcing non-root is the deployer's job via Pod Security Admission, not a Camel K bug. |

This is not build hygiene — it is the recognition that the answers in §4.8 /
§4.9 differ by build strategy and install scope. **The insecure-default
case is resolved** *(maintainer, 2026-05-18 — Q9)*: `BUILD_STRATEGY=routine`
is supported **only when CR authors are trusted**; exposing Camel K to
untrusted or multi-tenant CR authors **requires the `pod` build strategy**
(and §4.10 says so). A build-time impact on the operator pod under
`routine` with untrusted authors is therefore
`OUT-OF-MODEL: non-default-build`, not `VALID`.

---

## 4.6 Assumptions about inputs

Inputs arrive as Custom Resources via the Kube API (not as function
arguments). Per-"parameter" trust table — rows are CR kinds / fields /
client inputs:

| Entry point | Field / input | Attacker-controllable? | Caller (platform admin / cluster) must enforce |
| --- | --- | --- | --- |
| `Integration` | `spec.sources`, `spec.flows` | **Yes** — arbitrary route code, executed in the workload pod | RBAC: only principals trusted to run code in the namespace may create it |
| `Integration` | `spec.dependencies`, `spec.repositories` | **Yes** — arbitrary Maven GAVs / repo URLs, fetched + plugins run at build | Trusted Maven proxy/mirror; build strategy choice (§4.5a) |
| `Integration` | `spec.git` | **Yes** — arbitrary git URL cloned + Maven-built in-cluster | Same as dependencies |
| `Integration`/`builder` trait | `tasks` (`name;image;command`) | **Yes** — arbitrary image+command in builder pod (`Pod` strategy) | Treat CR-create as code-exec; isolate builds |
| `Integration`/`container` trait | `image`, `runAsUser`, `capabilitiesAdd` | **Yes** — arbitrary image / added capabilities | Pod Security Admission / policy engine at the namespace |
| `Integration` | `spec.template` (`pod` trait) | **Yes** — raw pod-spec strategic-merge (containers, volumes, securityContext) | Pod Security Admission; restrict the trait |
| `IntegrationKit` | `spec.image` | **Yes** — arbitrary image used directly | RBAC on IntegrationKit |
| `Kamelet` | `spec.template`, `spec.dependencies` | **Yes** — route logic injected into any Integration referencing it | RBAC on Kamelet; trust of remote Kamelet catalogs |
| `Pipe` | `spec.source`/`spec.sink` | **Yes** — generates an Integration | RBAC on Pipe |
| `IntegrationPlatform` / `IntegrationProfile` | `spec.build.*` (registry, baseImage, maven, strategy) | **Admin-only by intent** — platform-wide | Restrict RBAC to platform admins (§4.10) |
| `kamel run` | `http`/`github`/`gist` source URL | Yes — but resolved on the *user's* machine with the user's trust | User's own responsibility |
| Operator | Maven repo / registry / GitHub responses | Yes if the configured source is hostile | Choose trusted sources (§4.9, §4.10) |

Size/shape: CR shape is validated by CRD OpenAPI schema at the API server
plus ad-hoc Go validation in trait `Configure()`; **semantic safety of
images, GAVs, git URLs, Maven repos, and `builder.tasks` commands is not
validated — it is trusted** *(maintainer, 2026-05-18 — Q7: no admission
webhook, by design)*.

---

## 4.7 Adversary model

**In scope:**

- A **cluster tenant without RBAC to create Camel K CRs** who tries to
  cause builds/workloads anyway, or to read another tenant's
  Secrets/artifacts through Camel K. The model expects Kube RBAC to stop
  them and Camel K not to provide a bypass *(maintainer, 2026-05-18 —
  Q8)*.
- A principal who can create a Camel K CR in **namespace A** and thereby
  causes the operator to act in **namespace B** or **cluster-scope** in a
  way the principal could not do directly (confused-deputy / privilege
  escalation via the operator's broad RBAC, incl. `rolebindings`,
  `clusterrolebindings`, `pods/exec`). **This is IN SCOPE and a violation
  is `VALID`** *(maintainer, 2026-05-18 — Q4: a CR author is trusted only
  at their own namespace's level; any path from a namespace-A CR to
  rights in namespace B / cluster-scope the author lacked is a finding)*.
- A network client hitting the **operator's** health/metrics ports.

**Out of scope:**

- A principal who already has RBAC to create the CR in the namespace where
  the effect lands — they are trusted to run code there; "they have already
  won" at that trust level (§4.3).
- Platform admin / `IntegrationPlatform` editor / operator-Deployment
  editor (§4.3).
- A hostile Maven repo/registry/base image/Kamelet-catalog that the
  platform admin or CR author *chose* to point at (§4.3, §4.9).
- Attacks on the deployed route's own endpoints (Apache Camel core's
  model, §4.3).

**Multi-tenant note.** Camel K does not, by itself, provide tenant
isolation. A "tenant" boundary is only as strong as the Kubernetes RBAC,
namespace, Pod Security Admission, and (un-shipped) NetworkPolicy
configuration the platform admin applies around it *(maintainer,
2026-05-18 — Q12)*.

---

## 4.8 Security properties Camel K provides

Each property: statement + conditions, violation symptom, severity tier,
provenance. All properties below are **ratified** (maintainer, 2026-05-18).

1. **RBAC is the only gate, and Camel K does not weaken it.** The operator
   acts on a CR only if the Kube API admitted it; Camel K adds no
   authentication bypass of its own. *Violation symptom:* a principal
   causes a build/workload for a CR the API would have denied them, or the
   operator acts on a forged/cross-namespace CR. *Severity:*
   security-critical. *(maintainer, 2026-05-18 — Q8)*
2. **No cross-namespace / cross-trust escalation.** A CR author is trusted
   *only* at their own namespace's level; creating a CR in namespace A must
   not let the author obtain, via the operator, rights in namespace B or
   cluster-scope they did not already hold. *Violation symptom:* a CR
   author reads B's Secrets, binds themselves a Role/ClusterRole, or execs
   into a pod they could not otherwise. *Severity:* security-critical — a
   violation is `VALID` (not a `MODEL-GAP`). *(maintainer, 2026-05-18 —
   Q4)*
3. **Multi-operator non-contention.** Distinct operators reconcile only
   CRs annotated with their operator id; default operator also reconciles
   un-annotated CRs. *Violation symptom:* two operators fight over one CR
   (undefined behavior). *Severity:* correctness-only. *(documented —
   multi.adoc; feature deprecated)*
4. **Base image integrity by pinning.** The default builder/base image is
   pinned by digest (`eclipse-temurin:17-jdk@sha256:…`). *Violation
   symptom:* a different image is silently used at the pinned reference.
   *Severity:* security-critical (supply chain). *(documented —
   `defaults.go`)*
5. **Namespaced install confines the operator's Kube RBAC** to its
   namespace (Role, not ClusterRole). *Violation symptom:* a
   namespaced-install operator affects another namespace. *Severity:*
   security-critical. *(documented — `pkg/resources/config/rbac`;
   in-scope-as-a-property maintainer, 2026-05-18 — Q4)*
6. **Default pod hardening (partial).** Integration pods get
   `seccompProfile=RuntimeDefault` and (via the container trait) dropped
   capabilities by default. *Violation symptom:* pods run without these.
   *Severity:* hardening / correctness. **Note `runAsNonRoot` defaults to
   `false`** (§4.5a, §4.9) — by design, not a guarantee. *(documented —
   security-context.adoc for seccomp/runAsNonRoot; cap-drop + the
   not-a-guarantee stance maintainer, 2026-05-18 — Q11)*

**Resource properties:** Camel K bounds concurrent builds
(`MAX_RUNNING_BUILDS`, default 3 routine / 10 pod) and build wall-clock
(`BUILD_TIMEOUT_SECONDS`, default 300) *(documented — builds.adoc)*. It
makes **no** guarantee about memory/CPU of user builds or running
integrations beyond those knobs and any Kubernetes
LimitRange/ResourceQuota the admin sets — *(maintainer, 2026-05-18 —
Q13: no Camel K resource guarantee beyond those two knobs)*.

No confidentiality/integrity/availability or cryptographic properties are
claimed for Camel K itself (it is a control plane, not a crypto component).

---

## 4.9 Security properties Camel K does *not* provide

This is the most important section for an integrator.

- **No sandboxing of integration code.** A submitted Integration runs as a
  normal pod with the privileges its spec/traits request. Camel K is **not
  a security boundary between the CR author and the cluster** — submitting
  an Integration is, by design, arbitrary code execution in the target
  namespace. *(maintainer, 2026-05-18 — Q2)*
- **No isolation of the build from the operator (default strategy).** With
  `BUILD_STRATEGY=routine` (the default), untrusted Maven dependencies,
  git sources, and Maven settings are fetched and Maven — including
  arbitrary build plugins — is executed **inside the operator pod**, under
  the operator's ServiceAccount. The build is not sandboxed away from the
  operator's own privileges. **`routine` is supported only for trusted CR
  authors; untrusted/multi-tenant use requires the `pod` strategy** (§4.5a,
  §4.10). *(maintainer, 2026-05-18 — Q9)*
- **No supply-chain verification.** Camel K does not verify signatures or
  provenance of Maven artifacts, the container registry, the base image
  (beyond its own pinned default), or remote (GitHub) Kamelet catalogs.
  Whatever those sources return is built and run; choosing trusted sources
  is wholly the deployer's job. *(maintainer, 2026-05-18 — Q5)*
- **`builder.tasks`, `container.image`, the `pod` trait, and
  `spec.template`** are *by design* direct controls over images, commands,
  and raw pod spec run in-cluster. They are not vulnerabilities; they are
  the product. *(maintainer, 2026-05-18 — Q2)*
- **No multi-tenant isolation by itself.** Cross-tenant isolation is only
  what the surrounding Kube RBAC / namespaces / Pod Security Admission /
  NetworkPolicy provide. Camel K ships no NetworkPolicy. *(maintainer,
  2026-05-18 — Q12)*
- **No defense for the deployed route.** Camel K does not add
  authentication, TLS, header filtering, or rate limiting to the
  integration it deploys; the route's exposure — including DoS via
  resource exhaustion — is the route author's / operator's problem under
  Apache Camel core's model. *(documented — Camel Security Model: untrusted
  external message senders are Camel's "primary attacker model", and "DoS
  via resource exhaustion" is operator responsibility (Out of scope);
  `camel.apache.org/manual/security-model.html`. Camel-K boundary
  application: maintainer, 2026-05-18 — Q3)*

**False friends** (features that look like a security property but are not):

- *"Camel K / `kamel` secured my integration."* It does not. Camel K
  **deploys what you give it**; the only gate is Kube RBAC. The
  `security-context` trait sets a pod SecurityContext but **defaults
  `runAsNonRoot=false`** — present-but-permissive, not a guarantee
  *(documented — security-context.adoc; stance maintainer, 2026-05-18 —
  Q11)*.
- *"The operator validates my Integration, so bad input is rejected."* CRD
  schema + trait `Configure()` validate *shape*, not the *safety* of
  images/GAVs/git/commands. There is no admission webhook, by design.
  *(maintainer, 2026-05-18 — Q7)*
- *"Pinned base-image digest means the image supply chain is verified."*
  Only the default base image is pinned; user-chosen base images,
  dependencies, and `container.image` are not. *(maintainer, 2026-05-18 —
  Q5)*

**Well-known attack classes left to the operator/route author:** Maven
dependency-confusion / typosquatting and malicious build plugins; container
base-image and registry compromise; remote-Kamelet-catalog tampering; the
Apache Camel-core header-injection / message-injection class on the
deployed route (e.g. CVE-2025-27636) if the route is exposed to untrusted
networks. One sentence each — the point is to put the integrator on notice;
defending these is §4.10, not a Camel K bug.

---

## 4.10 Downstream responsibilities

For the **platform admin / operator-deployer** (and, for the last item, the
CR author):

1. **Treat "create/patch Integration / Pipe / Kamelet / IntegrationKit /
   Build / IntegrationProfile in namespace N" as equivalent to "run
   arbitrary code in N."** Grant that RBAC only to principals you trust at
   that level. *(maintainer, 2026-05-18 — Q2)*
2. **Lock down `IntegrationPlatform` / `IntegrationProfile` / the install
   namespace.** Whoever can edit them controls base image, registry, Maven
   settings, and build strategy for *every* integration. *(maintainer,
   2026-05-18 — Q4)*
3. **Choose the build strategy deliberately.** For untrusted or
   multi-tenant CR authors, do not run builds in the operator pod
   (`routine`); use `pod` strategy (and namespaced operators) so a
   malicious build cannot share the operator's ServiceAccount. **For
   untrusted/multi-tenant CR authors the `pod` strategy is required, not
   advisory.** *(maintainer, 2026-05-18 — Q9)*
4. **Own the supply chain.** Provide a trusted Maven proxy/mirror, a
   controlled registry, and trusted Kamelet sources; do not enable
   `insecure`/`InsecureSkipVerify` in production. *(documented —
   builds.adoc maven-proxy; ownership maintainer, 2026-05-18 — Q5)*
5. **Apply Kubernetes-native isolation around Camel K**: namespaces, Pod
   Security Admission, ResourceQuota/LimitRange, NetworkPolicies (Camel K
   ships none). *(maintainer, 2026-05-18 — Q12)*
6. **Understand the operator's RBAC before global install.** A global
   operator can create ClusterRoleBindings, ServiceAccounts, and `exec`
   into pods cluster-wide. *(maintainer, 2026-05-18 — Q4)*
7. **Set `runAsNonRoot` / `runAsUser` explicitly** if your cluster
   requires non-root; the default does not. *(documented —
   security-context.adoc; stance maintainer, 2026-05-18 — Q11)*
8. **(CR author / route author)** Apply Apache Camel core's endpoint
   hardening (header filtering, auth, TLS) to any route exposed to
   untrusted networks. *(documented — Camel security features catalog,
   `camel.apache.org/manual/security.html`)*

---

## 4.11 Known misuse patterns

- **Granting `Integration` create to untrusted users as a "low-privilege
  config submission."** It is arbitrary code execution. *Do instead:* gate
  it like `pods/create`; use namespaces + Pod Security Admission.
- **Multi-tenant on a single global operator with `routine` builds,
  expecting isolation.** Builds and the operator share a pod/SA. *Do
  instead:* per-tenant namespaced operators + `pod` build strategy
  (**required** for untrusted authors per Q9, 2026-05-18).
- **Believing Camel K hardens the deployed integration.** It deploys what
  you give it; `runAsNonRoot` defaults false. *Do instead:* set the
  security-context/container traits explicitly and enforce Pod Security
  Admission.
- **Pointing the platform at an unverified Maven repo / registry / remote
  Kamelet catalog and treating built images as trusted.** *Do instead:*
  trusted proxy/mirror, signed/pinned images, controlled Kamelet sources.
- **Using `container.image` / `builder.tasks` with untrusted images
  expecting the operator to vet them.** It does not. *Do instead:* restrict
  these traits via policy; vet images out of band.
- **Exposing the integration's endpoints to the internet without Camel
  core endpoint hardening.** *Do instead:* apply Camel's header
  filter/auth/TLS (the CVE-2025-27636 class).

---

## 4.11a Known non-findings (recurring false positives)

Feed this back as a suppression / negative-prompt list for scanners and AI
triage:

- **"The operator has powerful RBAC — can create RoleBindings /
  ClusterRoleBindings, ServiceAccounts, and `pods/exec`."** By design; it
  is how integrations are materialized (§4.5/§4.8). *Only* a finding if a
  principal **without** Camel K CR RBAC can leverage it, **or if it
  escalates a CR author beyond their own namespace's trust level — that
  case IS a finding and is `VALID`** (maintainer, 2026-05-18 — Q4; §4.8
  property 2). Per-permission noise on the documented RBAC is not.
- **"An Integration/Kamelet/Build runs arbitrary code, pulls any image,
  contacts any host, or runs Maven from the internet."** By design; the CR
  author is trusted at the namespace level and the supply chain is the
  deployer's (§4.7, §4.9). `BY-DESIGN: property-disclaimed`.
- **"`registry.insecure` / Maven `InsecureSkipVerify` disables TLS."**
  Opt-in, non-default knob, unsupported in production →
  `OUT-OF-MODEL: non-default-build` (maintainer, 2026-05-18 — Q5; §4.5a).
- **"Integration pods can run as root (`runAsNonRoot=false`)."** Documented
  default (§4.5a/§4.9); enforcement is the deployer's via Pod Security
  Admission → `BY-DESIGN` (maintainer, 2026-05-18 — Q11).
- **Static-analysis / scanner hits in `examples/`, `e2e/`, `proposals/`,
  sample CRs, or the deprecated Buildah/Kaniko/Spectrum tasks / `pod`
  trait / synthetic-Integration / multi-operator code.** `OUT-OF-MODEL:
  unsupported-component` (§4.3; maintainer, 2026-05-18 — Q6).
- **"`kamel` resolves remote `http`/`github`/`gist` sources."** Resolved on
  the invoking user's machine with the user's own trust — not an operator
  boundary.

---

## 4.12 Conditions that would change this model

- A new public CRD or a new CR field that influences build/exec.
- Addition (or removal) of an admission/validation/mutating/conversion
  webhook — would move or create the §4.4 boundary.
- A change of the default `BUILD_STRATEGY`, default publish strategy, or
  default `runAsNonRoot`.
- A change to the operator's RBAC (especially `rolebindings`,
  `clusterrolebindings`, `pods/exec`, `secrets`).
- Promotion of a deprecated/shipped-but-unsupported component (synthetic
  Integrations, multi-operator) into the supported product, or shipping a
  NetworkPolicy / sandboxing feature.
- Camel K gaining its own network listener beyond health/metrics.
- **Evidence the model is incomplete:** any report that cannot be cleanly
  routed to one §4.13 disposition is a `MODEL-GAP` and triggers a revision
  of §4.8/§4.9 — not an ad-hoc call.

---

## 4.13 Triage dispositions

| Disposition | Meaning | Licensed by |
| --- | --- | --- |
| `VALID` | Violates a §4.8 property via an in-scope adversary (§4.7) and an attacker-controllable input (§4.6). | §4.8, §4.6, §4.7 |
| `VALID-HARDENING` | No §4.8 property violated, but a §4.11 misuse is easy enough that the project elects to harden (e.g. safer default, opt-in policy). Maintainer discretion; typically no CVE. | §4.11 |
| `OUT-OF-MODEL: trusted-input` | Requires RBAC to create/patch the triggering CR, or admin control of `IntegrationPlatform` — a trusted role per §4.6/§4.2. | §4.6, §4.2 |
| `OUT-OF-MODEL: adversary-not-in-scope` | Requires a capability §4.7 excludes (platform admin, chosen-hostile supply chain). | §4.7 |
| `OUT-OF-MODEL: unsupported-component` | Lands in `examples/`, `e2e/`, `proposals/`, samples, or deprecated task/trait code. | §4.3 |
| `OUT-OF-MODEL: non-default-build` | Manifests only under a non-default/discouraged §4.5a knob (e.g. `routine` with untrusted authors, `insecure` TLS). | §4.5a |
| `BY-DESIGN: property-disclaimed` | Concerns a property §4.9 explicitly does not provide (sandboxing, build isolation under `routine`, supply-chain verification, route hardening). | §4.9 |
| `KNOWN-NON-FINDING` | Matches a §4.11a entry. | §4.11a |
| `MODEL-GAP` | Cannot be cleanly routed above. Triggers a §4.12 revision. (The operator-escalation case is **not** a gap — Q4 ratified it as `VALID`, §4.8 property 2.) | triggers §4.12 |

---

## 4.14 Maintainer ratification record

All open questions were ratified by a maintainer on **2026-05-18**. The
record is retained (per the threat-model rubric: keep provenance in the
published version so a closed report has a defensible citation —
*(maintainer, 2026-05-18)* — rather than bare prose). Body claims carry the
matching `Qn` tag. All questions below were confirmed as proposed.

- **Q1 — Intended use & non-goals — CONFIRMED.** Camel K is an in-cluster
  build-and-run platform; **not** a sandbox, a multi-tenant isolation
  boundary, or a supply-chain verifier. → §4.2, §4.3
- **Q2 — Central trust statement — CONFIRMED.** RBAC to create/patch any
  `camel.apache.org` CR is, by design, equivalent to arbitrary code +
  container execution in the target namespace; the only gate is Kube RBAC.
  → §4.2, §4.7, §4.9, §4.10
- **Q4 — Operator escalation boundary — CONFIRMED.** A CR author is
  trusted *only* at their own namespace's level; any path from a
  namespace-A CR to rights in namespace B / cluster-scope the author
  lacked is **in scope and `VALID`**. → §4.7, §4.8 (props 2, 5), §4.13
- **Q9 — `BUILD_STRATEGY=routine` — CONFIRMED.** Supported **only for
  trusted CR authors**; untrusted/multi-tenant use **requires** the `pod`
  strategy; build-time impact under `routine` with untrusted authors is
  `OUT-OF-MODEL: non-default-build`. → §4.5a, §4.9, §4.10, §4.13
- **Q3 — Division with Apache Camel core — CONFIRMED.** The deployed
  route's own runtime attack surface is Apache Camel core's threat model;
  Camel K's model ends at "image built and workload created." → §4.3,
  §4.9
- **Q5 — Supply-chain ownership — CONFIRMED.** Trust of Maven
  repos/registry/base image/remote Kamelet catalogs is wholly the
  deployer's; `insecure`/`InsecureSkipVerify` are non-default and
  unsupported in production. → §4.3, §4.5a, §4.9, §4.10
- **Q7 — No admission webhook by design — CONFIRMED.** The absence of any
  validating/mutating/conversion webhook is intentional; CRD schema +
  trait `Configure()` validate shape, not safety. → §4.4, §4.6, §4.9
- **Q8 — Kube API as sole authority — CONFIRMED.** Camel K never
  second-guesses or supplements Kube RBAC; "the API admitted the CR" =
  "authorized." → §4.5, §4.7, §4.8 (property 1)
- **Q10 — Negative side-effect inventory — CONFIRMED.** Aside from the
  Kube resources it manages and (under `routine`) exec-ing Maven in its
  own pod, the operator opens no listening sockets beyond health/metrics,
  installs no webhook, ships no NetworkPolicy. → §4.5
- **Q6 — Unsupported-code policy — CONFIRMED.** `examples/`, `e2e/`,
  `proposals/`, sample CRs, and the deprecated Buildah/Kaniko/Spectrum
  tasks + `pod` trait + synthetic Integrations + multi-operator/
  `OPERATOR_ID` — including the deprecated API surface still in the CRD —
  are out of the security model. → §4.3
- **Q11 — `runAsNonRoot=false` default — CONFIRMED.** Deliberate
  compatibility default; making pods non-root is the deployer's job via
  Pod Security Admission, not a Camel K bug. → §4.5a, §4.8 (prop 6), §4.9
- **Q12 — Multi-tenancy posture — CONFIRMED.** Camel K provides no tenant
  isolation itself; isolation is entirely Kube RBAC/namespace/PSA/
  NetworkPolicy configured by the admin. → §4.7, §4.9, §4.10
- **Q13 — Resource-exhaustion line — CONFIRMED.** The only resource
  guarantees are `MAX_RUNNING_BUILDS` and `BUILD_TIMEOUT_SECONDS`; CPU/mem
  is bounded only by Kubernetes quotas the admin sets — no Camel K
  guarantee. → §4.8

### Non-blocking meta (open — not a security-model claim)

- **Q-meta — Document coexistence & venue.** The Apache Camel **Security
  Model** is now published and reachable at
  `camel.apache.org/manual/security-model.html`. Its substance settles
  the *coexistence* question: it explicitly scopes itself to "Camel …
  embedded in someone else's application, **not a multi-tenant managed
  service**" and never addresses the Kubernetes operator / CR / cluster
  layer. Camel K is exactly that excluded layer, so this document is the
  **additive Camel-K sub-project expansion (option c)** — not a
  replacement of, nor folded into, the umbrella model, and not
  contradicting it (§4.1). What remains a **docs/PMC decision** is purely
  the *publication venue and linking*: where this lives
  (`docs/threat-model.md` is not published by the Antora site) and whether
  the Camel security pages should link to it. This does not gate triage
  and backs no claim. → §4.1

---

## 4.15 Machine-readable companion

A derived sidecar for automated/AI triage is emitted alongside this
document at `docs/threat-model.yaml`. The prose here is canonical; the
sidecar is regenerated whenever this file changes.

---

### Appendix — provenance back-map (source → §)

| Source | Lands in § |
| --- | --- |
| `README.adoc`, `architecture.adoc`, `operator.adoc` | §4.1, §4.2 |
| `installation/builds.adoc` | §4.5a, §4.8, §4.9, §4.10 |
| `traits/security-context.adoc`, `pkg/trait/security_context.go` | §4.5a, §4.8, §4.9 |
| `installation/advanced/multi.adoc` | §4.3, §4.8 (prop 3) |
| `pkg/resources/config/rbac/*` | §4.5, §4.7, §4.8, §4.10 |
| `pkg/util/defaults/defaults.go`, `pkg/platform/defaults.go` | §4.1, §4.5a, §4.8 (prop 4) |
| `security.apache.org/projects/` (ASF index lists only "Apache Camel") | §4.1 |
| Camel Security Model — "Camel … embedded in someone else's application, **not a multi-tenant managed service**" | §4.1 (this doc = the Camel-K expansion for the excluded layer), §4.14 Q-meta |
| Camel Security Model — route authors & deployment operators are "fully trusted"; "Code execution by a route author is by design and is not a vulnerability" | §4.2, §4.3, §4.9 (central trust statement / route-runtime division) |
| Camel Security Model — external message senders are the "primary attacker model" | §4.3, §4.9 (deployed-route attack surface is Camel core's) |
| Camel Security Model — "Out of scope": DoS via resource exhaustion is operator responsibility; documented opt-in insecure options | §4.9, §4.10, §4.5a (`insecure`/`InsecureSkipVerify`) |
| Maintainer ratification, 2026-05-18 | §4.14 and all *(maintainer)* tags |

> **Self-check status:** every section is substantive; no audit/code-review
> bullets; provenance tagged with only the three sanctioned tags; §4.9 and
> §4.10 are at least as substantive as §4.8; §4.6 has a per-parameter
> table; §4.13 enumerates a closed disposition set. **Ratified by
> maintainer 2026-05-18 — status ACCEPTED; the model closes reports across
> all sections.** Only the non-blocking Q-meta (publication venue) remains
> open and backs no claim.
