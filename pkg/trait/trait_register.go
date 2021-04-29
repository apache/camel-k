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
	AddToTraits(newInitTrait)
	AddToTraits(newPlatformTrait)
	AddToTraits(newCamelTrait)
	AddToTraits(newOpenAPITrait)
	AddToTraits(newKnativeTrait)
	AddToTraits(newKameletsTrait)
	AddToTraits(newErrorHandlerTrait)
	AddToTraits(newDependenciesTrait)
	AddToTraits(newBuilderTrait)
	AddToTraits(newQuarkusTrait)
	AddToTraits(newEnvironmentTrait)
	AddToTraits(newDeployerTrait)
	AddToTraits(newCronTrait)
	AddToTraits(newDeploymentTrait)
	AddToTraits(newGarbageCollectorTrait)
	AddToTraits(newAffinityTrait)
	AddToTraits(newTolerationTrait)
	AddToTraits(newKnativeServiceTrait)
	AddToTraits(newServiceTrait)
	AddToTraits(newContainerTrait)
	AddToTraits(newPullSecretTrait)
	AddToTraits(newJolokiaTrait)
	AddToTraits(newPrometheusTrait)
	AddToTraits(newJvmTrait)
	AddToTraits(newRouteTrait)
	AddToTraits(newIstioTrait)
	AddToTraits(newIngressTrait)
	AddToTraits(newServiceBindingTrait)
	AddToTraits(newOwnerTrait)
	AddToTraits(newPdbTrait)
	AddToTraits(newLoggingTraitTrait)
}
