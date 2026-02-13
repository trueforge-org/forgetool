package kubectlcmds

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	certificatesv1 "k8s.io/api/certificates/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	approvePendingCSRsOnceFn = approvePendingCSRsOnce
	approvePendingSleepFn    = time.Sleep
	approveListCSRsFn        = func(clientset *kubernetes.Clientset) (int, []interface{}, error) {
		csrList, err := clientset.CertificatesV1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return 0, nil, err
		}
		items := make([]interface{}, 0, len(csrList.Items))
		for i := range csrList.Items {
			items = append(items, csrList.Items[i])
		}
		return len(csrList.Items), items, nil
	}
	approveCSRLoopFn = func(clientset *kubernetes.Clientset, csr interface{}) error {
		typed := csr.(certificatesv1.CertificateSigningRequest)
		return approveCSR(clientset, typed)
	}
)

// ApprovePendingCertificates approves pending CSRs
func ApprovePendingCertificates(clientset *kubernetes.Clientset, stopCh <-chan struct{}) {
	log.Info().Msg("Waiting to approve certificates...")

	for {
		select {
		case <-stopCh:
			log.Info().Msg("Stopping certificate approval...")
			return
		default:
			approvePendingCSRsOnceFn(clientset)
			approvePendingSleepFn(5 * time.Second)
		}
	}
}

func approvePendingCSRsOnce(clientset *kubernetes.Clientset) {
	log.Debug().Msg("Retrieving list of pending CSRs")
	count, items, err := approveListCSRsFn(clientset)
	if err != nil {
		log.Error().Err(err).Msg("Error getting CSRs")
		return
	}

	log.Debug().Msgf("Retrieved %d CSRs", count)
	for _, raw := range items {
		csr := raw.(certificatesv1.CertificateSigningRequest)
		log.Debug().Str("CSRName", csr.Name).Msg("Checking CSR for approval")
		if csr.Status.Conditions == nil || len(csr.Status.Conditions) == 0 {
			if err := approveCSRLoopFn(clientset, csr); err != nil {
				log.Error().Str("CSRName", csr.Name).Err(err).Msg("Error approving CSR")
			} else {
				log.Info().Str("CSRName", csr.Name).Msg("Approved CSR")
			}
			continue
		}
		log.Debug().Str("CSRName", csr.Name).Msg("CSR already has approval conditions")
	}
}
