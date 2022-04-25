# Camel-k PDB Trait

This is an example of a Pod Distruption Budget for an integration pod running on the cluster. 
You can set a  `minumum available` or a `maximum unavailable` pods, it all depends on your eviction policy.

 **Note:** This trait is best used on a multi-node cluster. Also, you can only use one trait command. Either a `minimum available` or a `maximum unavailable`

Example: 

    kamel run pdb.java --trait pdb.enable=true --trait pdb.min-available=2

This runs at least 2 pods no matter what resources is needed by your scheduler. Unless you do a forcefull pod delete.

    kamel run pdb.java --trait pdb.enabled=true --trait pdb.max-unavailable=1
In this pod disruption budget, we have set the max unavailable pod to 1. Whenever our eviction policy kicks in, it must respect our budjet to always have only one pod unavailable. So, if we have a 3 pods integration deployment, and our policy is evicting pods to free up resources, there would only be a maximum of 1 pod evicted out of our deployment.

## Using ModeLine

        $Kamel run pdbModeline.java

