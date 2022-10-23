
Things to add:

* Filters
* Inheritance
* Views, allowing to return a bunch of objects
* Mime encoding
* Response timing
* Proper logging
* Nested routes
* Context support
* Allow to receive random structs that are parsed from the input form encoded or json
* Documentation
* Website
* Move params to request params?
* Use our own ResponseWriter and Request
* Autogenerate a JS/TS client with types.
* DOCS
* Add Middlewares
* Assets
* Wildcard path elemnts /docs/*docspath (that can include slashes)





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