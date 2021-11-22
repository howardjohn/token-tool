package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/howardjohn/auto-gcp-metadata"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	aud = flag.String("aud", "", "audience")
	sa  = flag.String("sa", "default", "sa")
	ns  = flag.String("namespace", "", "namespace")
	dur = flag.Duration("duration", time.Hour, "duration")
	jwt = flag.String("jwt", "", "prefetch jwt token")
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const (
	Create   = "create"
	Federate = "federate"
	Access   = "access"
)

func main() {
	flag.Parse()
	if len(os.Args) == 1 {
		fatal(fmt.Errorf("must pass a subcommand [create, federate, access]"))
	}
	meta := autogcp.NewMetadata()
	command := os.Args[1]
	k8sToken := *jwt
	if command == Create || k8sToken == "" {
		cc, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).ClientConfig()
		fatal(err)
		client, err := kubernetes.NewForConfig(cc)
		if err != nil {
			panic(err)
		}

		a := *aud
		if a == "" && command == Create {
			a = "istio-ca"
		} else if a == "" {
			a = meta.ProjectName() + ".svc.id.goog"
		}

		namespace := *ns
		if namespace == "" {
			namespace = meta.Namespace()
		}
		seconds := int64(dur.Seconds())
		token := &authenticationv1.TokenRequest{
			Spec: authenticationv1.TokenRequestSpec{
				Audiences:         []string{a},
				ExpirationSeconds: &seconds,
				//BoundObjectRef: &authenticationv1.BoundObjectReference{
				//	Kind:       "Pod",
				//	APIVersion: "v1",
				//	Name:       n.Name,
				//	UID:        p.UID,
				//},
			},
		}
		tok, err := client.CoreV1().ServiceAccounts(namespace).CreateToken(context.Background(), *sa, token, metav1.CreateOptions{})
		if err != nil {
			fatal(fmt.Errorf("could not create a token for %v/%v: %v", *sa, namespace, err))
		}
		k8sToken = tok.Status.Token
	}

	if command == Create {
		fmt.Println(k8sToken)
		return
	}
}
