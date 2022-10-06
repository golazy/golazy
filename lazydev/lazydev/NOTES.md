
The server start and listens to a port:

That will be long lived.

If the backend works, it will serve from the backend.
If the backend does not work, it will serve from the frontend.


# Requirements

* If it does not compile it should show the error message.
* Once it compiles, it should display the page instantly.

How to do that?

# Listening http in parent and conditionally forwarding to child
on an http request:

if backend.running: proxy request.
if backend.not handle request.

*Cons*: child runs on http



# Listening socket and forward everything to child

*Cons*: If child does not work, nothing works.


# Parent and child share the same listener. Parent is only enable if the client compiles. 

*Pros:* Client have a real socket and request is not forwarded
*Cons:* Parent can't respond to requests.

# Parent only listens the tcp socket. Clients does everything. The client compiles, and if it works, it creates a new child and let the previous one dies

How does the parent wait the new child?


# Parent only listens and starts/restart the client on fs changes.

How does the parent display compilation errors?




A) Child handles HTTP (No response when down)
B) Parent handles HTTP (Ugly proxy)
C) Parent handles HTTP only when child is dead (More complex)

