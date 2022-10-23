

Router
* Responsable of routing
* Does not know about resources
* It routes paths to actions



ResourceRouter
* Instantiate the Resources given the Resource definition
* Build the resources
* Add recursivily the resources


Resource
* Takes a controller
* Create the routes
* Takes care of the params and filters
* Create the actions from the controller
* Conf is in ResourceDefinition


Action
