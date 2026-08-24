package errs

import "strconv"

// An RPCCode is a gRPC status code, held as the number it is sent as.
//
// It is a number rather than the type from the gRPC module so that the table in
// [Kind] can name a code without the toolkit's error package depending on gRPC.
// The values are the canonical ones, so a server that does import gRPC converts
// with codes.Code(kind.RPCCode()).
type RPCCode uint32

// The canonical gRPC status codes.
const (
	RPCOK                 RPCCode = 0
	RPCCanceled           RPCCode = 1
	RPCUnknown            RPCCode = 2
	RPCInvalidArgument    RPCCode = 3
	RPCDeadlineExceeded   RPCCode = 4
	RPCNotFound           RPCCode = 5
	RPCAlreadyExists      RPCCode = 6
	RPCPermissionDenied   RPCCode = 7
	RPCResourceExhausted  RPCCode = 8
	RPCFailedPrecondition RPCCode = 9
	RPCAborted            RPCCode = 10
	RPCOutOfRange         RPCCode = 11
	RPCUnimplemented      RPCCode = 12
	RPCInternal           RPCCode = 13
	RPCUnavailable        RPCCode = 14
	RPCDataLoss           RPCCode = 15
	RPCUnauthenticated    RPCCode = 16
)

var rpcNames = [...]string{
	RPCOK:                 "OK",
	RPCCanceled:           "Canceled",
	RPCUnknown:            "Unknown",
	RPCInvalidArgument:    "InvalidArgument",
	RPCDeadlineExceeded:   "DeadlineExceeded",
	RPCNotFound:           "NotFound",
	RPCAlreadyExists:      "AlreadyExists",
	RPCPermissionDenied:   "PermissionDenied",
	RPCResourceExhausted:  "ResourceExhausted",
	RPCFailedPrecondition: "FailedPrecondition",
	RPCAborted:            "Aborted",
	RPCOutOfRange:         "OutOfRange",
	RPCUnimplemented:      "Unimplemented",
	RPCInternal:           "Internal",
	RPCUnavailable:        "Unavailable",
	RPCDataLoss:           "DataLoss",
	RPCUnauthenticated:    "Unauthenticated",
}

// String is the code's name as gRPC spells it, so that a log line and a gRPC
// trace of the same call say the same word.
func (c RPCCode) String() string {
	if int(c) >= len(rpcNames) {
		return "Code(" + strconv.FormatUint(uint64(c), 10) + ")"
	}
	return rpcNames[c]
}
