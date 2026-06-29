// Package lazyforms provides model-aware form helpers for GoLazy views.
//
// The package is about rendering forms, not parsing submitted requests. It
// builds form actions, ids, classes, field names, field ids, labels, and input
// controls from Go structs. Field names and ids come from lazyschema, so the
// names emitted by these helpers are the names lazycontroller.Base.Decode later
// decodes from request form values.
//
// lazyapp installs these helpers automatically on the application's lazyview
// renderer after it creates the lazyroutes router. In a normal GoLazy app,
// templates can call helpers such as form_for, text_field, checkbox_field, and
// submit_button without any manual registration. form_for uses the router to
// choose create, update, or delete paths for models, renders the model's form
// partial, and makes the current form available to nested field helpers.
//
// Applications that assemble lazyview directly can install the helpers with:
//
//	views.AddHelpers(lazyforms.Helpers(router))
//
// The router only needs to satisfy Router for model actions. If form_route is
// used, the router must also provide PathFor, as lazyroutes.Scope does. When no
// route or router path is available, forms render with action "#".
//
// A model that implements Resource is treated as persisted when Persisted
// returns true. Persisted forms use POST plus a hidden _method value of "patch"
// by default, and delete_button_for uses POST plus hidden _method "delete".
// Non-persisted models use POST without an override. NumericID and StringID are
// optional fallbacks for edit form ids when RouteParam is not available.
//
// The default form partial is based on the model name, for example a Car model
// renders "car_form" and a RaceCar model renders "race_car_form". form_file can
// override that partial, form_model and form_scope can override the form scope,
// and form_multipart adds multipart/form-data for file uploads.
//
// Field helpers can be used inside an active form_for partial by passing only a
// Go field path, such as "Title" or "Author.Name". They can also receive an
// explicit *Form as the first argument when a template needs to render fields
// outside the active form body. Struct tags understood by lazyschema affect the
// generated names; fields tagged schema:"-" or form:"-" are skipped by
// form_fields.
package lazyforms
