package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the kubernetes clientset with lazy initialization.
type Client struct {
	once       sync.Once
	clientset  *kubernetes.Clientset
	err        error
	kubeconfig string
}

// NewClient creates a Client that will initialize the clientset on first use.
func NewClient(kubeconfig string) *Client {
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	return &Client{kubeconfig: kubeconfig}
}

// Clientset returns the initialized kubernetes.Clientset, initializing it on first call.
func (c *Client) Clientset() (*kubernetes.Clientset, error) {
	c.once.Do(func() {
		cfg, err := clientcmd.BuildConfigFromFlags("", c.kubeconfig)
		if err != nil {
			c.err = fmt.Errorf("building kubeconfig: %w", err)
			return
		}
		c.clientset, c.err = kubernetes.NewForConfig(cfg)
	})
	return c.clientset, c.err
}
