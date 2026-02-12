package kubectlcmds

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
			approvePendingCSRsOnce(clientset)
			time.Sleep(5 * time.Second)
		}
	}
}

func approvePendingCSRsOnce(clientset *kubernetes.Clientset) {
	log.Debug().Msg("Retrieving list of pending CSRs")
	csrList, err := clientset.CertificatesV1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error getting CSRs")
		return
	}

	log.Debug().Msgf("Retrieved %d CSRs", len(csrList.Items))
	for _, csr := range csrList.Items {
		log.Debug().Str("CSRName", csr.Name).Msg("Checking CSR for approval")
		if csr.Status.Conditions == nil || len(csr.Status.Conditions) == 0 {
			if err := approveCSR(clientset, csr); err != nil {
				log.Error().Str("CSRName", csr.Name).Err(err).Msg("Error approving CSR")
			} else {
				log.Info().Str("CSRName", csr.Name).Msg("Approved CSR")
			}
			continue
		}
		log.Debug().Str("CSRName", csr.Name).Msg("CSR already has approval conditions")
	}
}
