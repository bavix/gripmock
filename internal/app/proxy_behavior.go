package app

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
)

type proxyBehavior interface {
	proxyOnly() bool
	captureMiss() bool
	canFallback(err error) bool
}

type proxyOnlyBehavior struct{}

func (proxyOnlyBehavior) proxyOnly() bool            { return true }
func (proxyOnlyBehavior) captureMiss() bool          { return false }
func (proxyOnlyBehavior) canFallback(err error) bool { return false }

type replayBehavior struct{}

func (replayBehavior) proxyOnly() bool            { return false }
func (replayBehavior) captureMiss() bool          { return false }
func (replayBehavior) canFallback(err error) bool { return status.Code(err) == codes.NotFound }

type captureBehavior struct{}

func (captureBehavior) proxyOnly() bool            { return false }
func (captureBehavior) captureMiss() bool          { return true }
func (captureBehavior) canFallback(err error) bool { return status.Code(err) == codes.NotFound }

//nolint:ireturn
func newProxyBehavior(route *proxyroutes.Route) proxyBehavior {
	if route == nil {
		return nil
	}

	switch route.Mode {
	case proxyroutes.ModeProxy:
		return proxyOnlyBehavior{}
	case proxyroutes.ModeCapture:
		return captureBehavior{}
	case proxyroutes.ModeReplay:
		return replayBehavior{}
	default:
		return proxyOnlyBehavior{}
	}
}
