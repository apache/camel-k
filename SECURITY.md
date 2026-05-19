# Security Policy

## Supported Versions

Apache Camel K is released and supported under the Apache Camel umbrella. For the currently
supported versions and their compatibility, see the
[Camel K release page](https://camel.apache.org/camel-k/next/installation/installation.html)
and the [GitHub releases](https://github.com/apache/camel-k/releases).

## Reporting a Vulnerability

Apache Camel K follows the Apache Software Foundation security process. For information on how
to report a new security problem please see [here](https://camel.apache.org/security/).

**Do not** open a public GitHub issue, pull request, or discussion for a suspected
vulnerability — report it privately through the ASF process above (security@apache.org).

## Security Model

Before submitting a report, please read the project's
[Threat Model](docs/threat-model.md). It documents who is trusted, where the trust boundaries
sit, which findings count as a Camel K vulnerability, and which categories are out of scope —
e.g. a CR author running code in their own namespace (by design), platform-admin actions, a
hostile Maven repository / registry / base image / remote Kamelet catalog the deployer chose
to trust, the deployed route's own runtime behaviour (Apache Camel core's model), behaviour
reachable only under a discouraged non-default knob, and shipped-but-unsupported code.

The Camel K threat model is the additive sub-project expansion of the umbrella
[Apache Camel Security Model](https://camel.apache.org/manual/security-model.html), which
explicitly scopes itself to "Camel embedded in someone else's application, not a multi-tenant
managed service"; the Kubernetes operator / Custom Resource / cluster layer is documented by
Camel K's own threat model. A `docs/threat-model.yaml` sidecar mirrors the triage-relevant
facts for automated tooling.

Reports outside the documented scope will be closed with a reference to that document.
