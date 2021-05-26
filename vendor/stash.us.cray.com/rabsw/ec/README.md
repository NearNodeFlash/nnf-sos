# Element Controller
Defines an Element Controller (EC) that is capable of self-hosting an http or grpc server and handling requests as defined by Element Controller's Routers.

An EC can be Run in one of two modes:
*  As a GRPC Server. This is the default behavior when an EC is run as a service to the dp-api. The Send() method can be used to send an http.Request to the EC.
*  As a standard HTTP Server. This is useful during testing of an EC when the dp-api is not providing the http frontend.

An EC can also be Attached to an existing *mux.Router with a specific mode defining how to handle the Routes:
*  Forward All. This is the desired behavior when a EC is configured for GRPC and all routes handled by the EC should be forwarded over GRPC. Each Route is sent over GRPC to a listening GRPC server.
*  Reject All. This is the desired behavior when all routes defined by the EC should be rejected. A http.NotImplementedError is returned. Use this mode when the EC is disabled.

## Routers & Routes
Routers are identified by a unique name and define initialize/start routines that can be used for router configuration and runtime behavior. Routers also define Routes, which are http endpoints handled by the Router. Routes contain a user friendly name, the http.Methods, the Path with regex notation, and a handler function to execute when the route is received. 

```go
    {
        Name:        "RedfishV1FabricsFabricIdGet",
        Method:      ec.GET_METHOD,
        Path:        "/redfish/v1/Fabrics/{FabricId}",
        HandlerFunc: RedfishV1FabricsFabricIdGet,
    }
```

In the above example, the HandlerFunc `RedfishV1FabricsFabricIdGet` is called when any GET is received matching the `"/redfish/v1/Fabrics/{FabricId}"` path, with `FabricId` being any wildcard string. Path regular expression is decoded by the router with the untility function `mux.Vars(r)` returning a dictionary with wildcards decoded. In the above, the fabricId can be accessed with `mux.Vars(r)['FabricId']`
