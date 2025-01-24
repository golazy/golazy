package lazydispatch

//type Router struct {
//	Routes     []*Route
//	dispatcher *Dispatcher
//	httpr      *router.Router[Route]
//	names      *namedRoutes
//}
//
//func NewRouter() *Router {
//	router := &Router{
//		httpr: router.NewRouter[Route](),
//		names: NewNamedRoutes(),
//	}
//	router.dispatcher = New(http.HandlerFunc(router.app))
//
//	return router
//}
//
//func (r *Router) app(w http.ResponseWriter, req *http.Request) {
//	h := r.httpr.Find(req)
//	if h == nil {
//		http.NotFound(w, req)
//		return
//	}
//	req = req.WithContext(context.WithValue(req.Context(), "*router.Route", r))
//	(*h).Handler.ServeHTTP(w, req)
//}
//
//
//var verbs = lazysupport.NewStringSet("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD")
//
