/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trait

func init() {
	// List of default trait factories.
	// Declaration order is not important, but let's keep them sorted for debugging.
	AddToTraits(newAffinityTrait)
	AddToTraits(newBuilderTrait)
	AddToTraits(newCamelTrait)
	AddToTraits(newConfigurationTrait)
	AddToTraits(newContainerTrait)
	AddToTraits(newCronTrait)
	AddToTraits(newDependenciesTrait)
	AddToTraits(newDeployerTrait)
	AddToTraits(newDeploymentTrait)
	AddToTraits(newEnvironmentTrait)
	AddToTraits(newErrorHandlerTrait)
	AddToTraits(newGarbageCollectorTrait)
	AddToTraits(newIngressTrait)
	AddToTraits(newIstioTrait)
	AddToTraits(newJolokiaTrait)
	AddToTraits(newJvmTrait)
	AddToTraits(newKameletsTrait)
	AddToTraits(newKnativeTrait)
	AddToTraits(newKnativeServiceTrait)
	AddToTraits(newLoggingTraitTrait)
	AddToTraits(newInitTrait)
	AddToTraits(newOpenAPITrait)
	AddToTraits(newOwnerTrait)
	AddToTraits(newPdbTrait)
	AddToTraits(newPlatformTrait)
	AddToTraits(newPodTrait)
	AddToTraits(newPrometheusTrait)
	AddToTraits(newPullSecretTrait)
	AddToTraits(newQuarkusTrait)
	AddToTraits(newRouteTrait)
	AddToTraits(newServiceTrait)
	AddToTraits(newServiceBindingTrait)
	AddToTraits(newTolerationTrait)
	// ^^ Declaration order is not important, but let's keep them sorted for debugging.
}
