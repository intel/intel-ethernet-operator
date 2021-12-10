// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package drainhelper

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"os"
	"strconv"
	"time"

	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/kubectl/pkg/drain"
)

const (
	drainHelperTimeoutEnvVarName = "DRAIN_TIMEOUT_SECONDS"
	drainHelperTimeoutDefault    = int64(90)
	leaseDurationEnvVarName      = "LEASE_DURATION_SECONDS"
	leaseDurationDefault         = int64(600)
)

type DrainHelper struct {
	log       logr.Logger
	clientSet *clientset.Clientset
	nodeName  string

	drainer              *drain.Helper
	leaseLock            *resourcelock.LeaseLock
	leaderElectionConfig leaderelection.LeaderElectionConfig
}

func getOsVarOrUseDefault(log logr.Logger, varName string, defVal int64) int64 {
	retValStr := os.Getenv(varName)

	if retValStr == "" {
		log.Info("env variable not found - using default value", "variable", varName, "default", defVal)
		return defVal
	}

	val, err := strconv.ParseInt(retValStr, 10, 64)

	if err != nil {
		log.Error(err, "failed to parse env variable to int64 - using default value", "variable", varName, "value", retValStr, "default", defVal)
		return defVal
	}

	return val
}

func NewDrainHelper(log logr.Logger, cs *clientset.Clientset, nodeName, namespace string) *DrainHelper {
	drainTimeout := getOsVarOrUseDefault(log, drainHelperTimeoutEnvVarName, drainHelperTimeoutDefault)
	log.Info("drain settings", "timeout seconds", drainTimeout)

	leaseDur := getOsVarOrUseDefault(log, leaseDurationEnvVarName, leaseDurationDefault)
	log.Info("lease settings", "duration seconds", leaseDur)

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "clv-daemon-lease",
			Namespace: namespace,
		},
		Client: cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: nodeName,
		},
	}

	return &DrainHelper{
		log:       log,
		clientSet: cs,
		nodeName:  nodeName,

		drainer: &drain.Helper{
			Client:              cs,
			Force:               true,
			IgnoreAllDaemonSets: true,
			DeleteEmptyDirData:  true,
			GracePeriodSeconds:  -1,
			Timeout:             time.Duration(drainTimeout) * time.Second,
			OnPodDeletedOrEvicted: func(pod *corev1.Pod, usingEviction bool) {
				act := "Deleted"
				if usingEviction {
					act = "Evicted"
				}
				log.Info("pod evicted or deleted", "action", act, "pod", fmt.Sprintf("%s/%s", pod.Name, pod.Namespace))
			},
			Out:    &utils.LogWriter{Log: log, Stream: "stdout"},
			ErrOut: &utils.LogWriter{Log: log, Stream: "stderr"},
		},

		leaseLock: lock,
		leaderElectionConfig: leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   time.Duration(leaseDur) * time.Second,
			RenewDeadline:   15 * time.Second,
			RetryPeriod:     5 * time.Second,
		},
	}
}

// Run joins leader election and drains(only if drain is set) the node if becomes a leader.
//
// f is a function that takes a context and returns a bool.
// It should return true if uncordon should be performed(Only applicable if drain is set to true).
// If `f` returns false, the uncordon does not take place. This is useful in 2-step scenario like fwddp-daemon where
// reboot must be performed without loosing the leadership and without the uncordon.
func (dh *DrainHelper) Run(f func(context.Context) bool, drain bool) error {
	defer func() {
		// Following mitigation is needed because of the bug in the leader election's release functionality
		// Release fails because the input (leader election record) is created incomplete (missing fields):
		// Failed to release lock: Lease.coordination.k8s.io "clv-daemon-lease" is invalid:
		// ... spec.leaseDurationSeconds: Invalid value: 0: must be greater than 0
		// When the leader election finishes (Run() ends), we need to clean up the Lease manually.
		// See: https://github.com/kubernetes/kubernetes/pull/80954
		// This however is not critical - if the leader will not refresh the lease,
		// another node will take it after some time.

		dh.log.Info("releasing the lock (bug mitigation)")

		leaderElectionRecord, _, err := dh.leaseLock.Get(context.Background())
		if err != nil {
			dh.log.Error(err, "failed to get the LeaderElectionRecord")
			return
		}
		leaderElectionRecord.HolderIdentity = ""
		if err := dh.leaseLock.Update(context.Background(), *leaderElectionRecord); err != nil {
			dh.log.Error(err, "failed to update the LeaderElectionRecord")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var innerErr error

	lec := dh.leaderElectionConfig
	lec.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			dh.log.Info("started leading")

			uncordonAndFreeLeadership := func() {
				// always try to uncordon the node
				// e.g. when cordoning succeeds, but draining fails
				dh.log.Info("uncordoning node")
				if err := dh.uncordon(ctx); err != nil {
					dh.log.Error(err, "uncordon failed")
					innerErr = err
				}
			}

			if drain {
				dh.log.Info("cordoning & draining node")
				if err := dh.cordonAndDrain(ctx); err != nil {
					dh.log.Error(err, "cordonAndDrain failed")
					innerErr = err
					uncordonAndFreeLeadership()
					return
				}
			}

			dh.log.Info("worker function - start")
			performUncordon := f(ctx)
			dh.log.Info("worker function - end", "performUncordon", performUncordon)
			if drain && performUncordon {
				uncordonAndFreeLeadership()
			}

			dh.log.Info("cancelling the context to finish the leadership")
			cancel()
		},
		OnStoppedLeading: func() {
			dh.log.Info("stopped leading")
		},
		OnNewLeader: func(id string) {
			if id != dh.nodeName {
				dh.log.Info("new leader elected", "this", dh.nodeName, "leader", id)
			}
		},
	}

	le, err := leaderelection.NewLeaderElector(lec)
	if err != nil {
		dh.log.Error(err, "failed to create new leader elector")
		return err
	}

	le.Run(ctx)

	if innerErr != nil {
		dh.log.Error(innerErr, "error during (un)cordon or drain actions")
	}

	return innerErr
}

func (dh *DrainHelper) cordonAndDrain(ctx context.Context) error {
	node, nodeGetErr := dh.clientSet.CoreV1().Nodes().Get(ctx, dh.nodeName, metav1.GetOptions{})
	if nodeGetErr != nil {
		dh.log.Error(nodeGetErr, "failed to get the node object")
		return nodeGetErr
	}

	var e error
	backoff := wait.Backoff{Steps: 5, Duration: 15 * time.Second, Factor: 2}
	f := func() (bool, error) {
		if err := drain.RunCordonOrUncordon(dh.drainer, node, true); err != nil {
			dh.log.Info("failed to cordon the node - retrying", "nodeName", dh.nodeName, "reason", err.Error())
			e = err
			return false, nil
		}

		if err := drain.RunNodeDrain(dh.drainer, dh.nodeName); err != nil {
			dh.log.Info("failed to drain the node - retrying", "nodeName", dh.nodeName, "reason", err.Error())
			e = err
			return false, nil
		}

		return true, nil
	}

	dh.log.Info("starting drain attempts")
	if err := wait.ExponentialBackoff(backoff, f); err != nil {
		if err == wait.ErrWaitTimeout {
			dh.log.Error(e, "failed to drain node - timed out")
			return e
		}
		dh.log.Error(err, "failed to drain node")
		return err
	}

	dh.log.Info("node drained")
	return nil
}

func (dh *DrainHelper) uncordon(ctx context.Context) error {
	node, err := dh.clientSet.CoreV1().Nodes().Get(ctx, dh.nodeName, metav1.GetOptions{})
	if err != nil {
		dh.log.Error(err, "failed to get the node object")
		return err
	}

	var e error
	backoff := wait.Backoff{Steps: 5, Duration: 15 * time.Second, Factor: 2}
	f := func() (bool, error) {
		if err := drain.RunCordonOrUncordon(dh.drainer, node, false); err != nil {
			dh.log.Error(err, "failed to uncordon the node - retrying", "nodeName", dh.nodeName)
			e = err
			return false, nil
		}

		return true, nil
	}

	dh.log.Info("starting uncordon attempts")
	if err := wait.ExponentialBackoff(backoff, f); err != nil {
		if err == wait.ErrWaitTimeout {
			dh.log.Error(e, "failed to uncordon node - timed out")
			return e
		}
		dh.log.Error(err, "failed to uncordon node")
		return err
	}
	dh.log.Info("node uncordoned")

	return nil
}
