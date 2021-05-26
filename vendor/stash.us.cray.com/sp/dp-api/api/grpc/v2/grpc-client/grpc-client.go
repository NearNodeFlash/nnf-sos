/* -----------------------------------------------------------------
 * grpc-client.go -
 *
 * Makes gRPC requests to the Element Controllers and processes their responses.
 *
 * Author: Tim Morneau
 *
 * © Copyright 2021 Hewlett Packard Enterprise Development LP
 *
 * ----------------------------------------------------------------- */

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	dpApiMsgs "stash.us.cray.com/sp/dp-common/api/proto/v2/dp-api_msgs"
	pb "stash.us.cray.com/sp/dp-common/api/proto/v2/dp-ec"
	sf "stash.us.cray.com/sp/rfsf-openapi/pkg/models"
)

const (
	// Version of API provided by DP-API service
	apiVersion = "v2"
)

type ClientRequest struct {
	ElmntCntrlName string
	ElmntCntrlPort string
	HTTPwriter     http.ResponseWriter
	HTTPrequest    *http.Request
	Error          error
}

// Maps gRPC status codes to HTTP status codes
// List of gRPC status codes: https://godoc.org/google.golang.org/grpc/codes#Code
// List of HTTP status codes: https://golang.org/src/net/http/status.go
var errorMap = map[codes.Code]int{
	codes.OK:                 http.StatusOK,                           // 200
	codes.Canceled:           http.StatusNoContent,                    // 204 - Client cancelled request, server should return no content
	codes.Unknown:            http.StatusInternalServerError,          // 500
	codes.InvalidArgument:    http.StatusBadRequest,                   // 400
	codes.DeadlineExceeded:   http.StatusRequestTimeout,               // 408
	codes.NotFound:           http.StatusNotFound,                     // 404
	codes.AlreadyExists:      http.StatusBadRequest,                   // 400
	codes.PermissionDenied:   http.StatusForbidden,                    // 403
	codes.ResourceExhausted:  http.StatusInsufficientStorage,          // 507 - The "resource" would most likely be storage
	codes.FailedPrecondition: http.StatusPreconditionFailed,           // 412
	codes.Aborted:            http.StatusInternalServerError,          // 500
	codes.OutOfRange:         http.StatusRequestedRangeNotSatisfiable, // 416
	codes.Unimplemented:      http.StatusMethodNotAllowed,             // 405
	codes.Internal:           http.StatusInternalServerError,          // 500
	codes.Unavailable:        http.StatusServiceUnavailable,           // 503
	codes.DataLoss:           http.StatusInternalServerError,          // 500
	codes.Unauthenticated:    http.StatusUnauthorized,                 // 401
}

// Sends a task request to be performed by the element controller.
// Writes the EC response to REST caller, or the error if one occurred.
func (grpcReq *ClientRequest) ProcessRequest(request *pb.ECTaskRequest) {

	// Set up a connection -- the service is running in a pod, so use localhost
	conn, err := grpc.Dial("localhost"+grpcReq.ElmntCntrlPort, grpc.WithInsecure())
	if err != nil {
		log.WithFields(log.Fields{"err": err, "Element Controller: ": grpcReq.ElmntCntrlName}).Warn("did not connect")
		grpcReq.respondError(err)
		return // Don't try to further process the request
	}
	defer conn.Close()

	// Establish deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Send request and capture response/any errors returned
	client := pb.NewControllerServiceClient(conn)
	if response, err := client.ProcessTaskRequest(ctx, request); err != nil {
		grpcReq.respondError(err)
	} else {
		grpcReq.respondSuccess(response.JsonData)
	}
}

// Checks to see if the response received from an EC is a Task Resource.
// If so, we return a 202 - Accepted instead of a 200 - OK.
func isRedfishTaskResponse(response string) bool {

	responseBytes := []byte(response)
	var model sf.Task

	// Unmarshal response string into Task model
	if err := json.Unmarshal(responseBytes, &model); err != nil {
		log.Warnf("Unable to unmarshal response into Task model: %v", err)
		return false
	} else { // No errors encountered during unmarshal, so check OdataType
		return model.OdataType == dpApiMsgs.TaskType
	}
}

/* Processes any errors returned from gRPC call to EC.
 * Maps gRPC errors to HTTP errors to be returned to the REST caller.
 * A gRPC Status represents an RPC status code, message, and details.
 * gRPC Status: https://godoc.org/google.golang.org/grpc/internal/status#Status
 */
func (grpcReq *ClientRequest) respondError(err error) {
	log.WithField("err", err).Error("Error returned from gRPC Client Request")
	var redfishError sf.RedfishError

	// Default error properties
	var message string
	var code int

	// Check if error returned was created from gRPC
	if errStatus, ok := status.FromError(err); ok {
		log.Info("Error is a valid gRPC prototype; mapping to appropriate HTTP status")
		code = errorMap[errStatus.Code()]
		message = errStatus.Message()
	} else {
		message = fmt.Sprintf("Internal Server Error: %v", err)
		code = http.StatusInternalServerError
		log.Warn("Error was not created by gRPC; returning generic internal server error")
	}

	redfishError.Error.Code = strconv.Itoa(code)
	redfishError.Error.Message = message

	// Marshal, format, and return RedfishError response
	var formattedJson bytes.Buffer
	response, _ := json.Marshal(redfishError)
	json.Indent(&formattedJson, response, "", "\t")
	grpcReq.respondJson(string(formattedJson.Bytes()), code)
}

func (grpcReq *ClientRequest) respondSuccess(jsonResponse string) {

	/* Determine the HTTP Status code to return.
	 	If the EC responded with a Task Resource, then
		the request was asynchronous, and we need to return a
		202 - Accepted response. Otherwise, we can return a
		200 - OK response.
	*/
	var httpStatus int
	if isRedfishTaskResponse(jsonResponse) {
		log.Info("Response is a Task Resource -- returning 202 Accepted")
		httpStatus = http.StatusAccepted
	} else {
		log.Info("Response is not a Task Resource -- returning 200 OK")
		httpStatus = http.StatusOK
	}

	grpcReq.respondJson(jsonResponse, httpStatus)
}

// Does the actual writing of the HTTP response
func (grpcReq *ClientRequest) respondJson(body string, status int) {
	log.Infof("Response body: %s", body)
	grpcReq.HTTPwriter.Header().Set("Content-Type", "application/json; charset=UTF-8")
	grpcReq.HTTPwriter.WriteHeader(status)
	grpcReq.HTTPwriter.Write([]byte(body))
}
