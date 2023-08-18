package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"gomodules.xyz/jsonpatch/v3"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	resourceName    corev1.ResourceName = "yusur.tech/sriov_netdevice"
	filterKey                           = "dpu.yusur.tech/status"
	filterVal                           = "true"
	multiCniAnnoKey                     = "k8s.v1.cni.cncf.io/networks"
	multiCniAnnoVal                     = "kube-system/yusur-cni-net@dpuvf"
	quatity                             = "1"
)

type AdmissionReview struct {
	Request  *AdmissionRequest  `json:"request,omitempty"`
	Response *AdmissionResponse `json:"response,omitempty"`
}

type AdmissionRequest struct {
	UID string `json:"uid,omitempty"`
}

type AdmissionResponse struct {
	UID     string `json:"uid,omitempty"`
	Allowed bool   `json:"allowed"`
}

func main() {
	var isTls bool
	flag.BoolVar(&isTls, "tls", false, "enable tls")
	flag.Parse()
	server := &http.Server{
		Addr: ":8443",
	}
	http.HandleFunc("/mutate", mutateHandler)
	http.HandleFunc("/sms", smsHandler)

	if isTls {
		// 加载TLS证书
		cert, err := tls.LoadX509KeyPair("webhook.crt", "webhook.key")
		if err != nil {
			log.Fatalf("Failed to load key pair: %v", err)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "")
		fmt.Println("Webhook server tls started")
		if err != nil {
			fmt.Println("==> err: ", err)
		}
	} else {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println("==> err: ", err)
		}
	}

}

func smsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("sms")
	w.Write([]byte("response"))
}

func mutateHandler(w http.ResponseWriter, r *http.Request) {
	klog.Info("/mutate\n")
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	klog.Infof("==> body: %s\n", body)
	ar := admissionv1.AdmissionReview{}

	err = json.Unmarshal(body, &ar)
	if err != nil {
		fmt.Printf("==> err: %v\n", err)
		http.Error(w, "Failed to unmarshal admission review", http.StatusBadRequest)
		return
	}

	pod := corev1.Pod{}
	err = json.Unmarshal(ar.Request.Object.Raw, &pod)
	if err != nil {
		http.Error(w, "Error marshalling original pod", http.StatusInternalServerError)
		return
	}
	q := resource.MustParse(quatity)
	var container *corev1.Container
	container = &pod.Spec.Containers[0]
	if container.Resources.Limits == nil {
		container.Resources.Limits = corev1.ResourceList{}
	}
	if _, ok := container.Resources.Limits[resourceName]; !ok {
		container.Resources.Limits[resourceName] = q
	}

	if container.Resources.Requests == nil {
		container.Resources.Requests = corev1.ResourceList{}
	}
	if _, ok := container.Resources.Requests[resourceName]; !ok {
		container.Resources.Requests[resourceName] = q
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if pod.Annotations[multiCniAnnoKey] != multiCniAnnoVal {
		pod.Annotations[multiCniAnnoKey] = multiCniAnnoVal
	}

	if err != nil {
		klog.Errorf("==> err: %v\n", err)
		http.Error(w, "Failed to unmarshal pod", http.StatusBadRequest)
		return
	}

	patchedPodBytes, err := json.Marshal(pod)
	if err != nil {
		klog.Errorf("==> err: %v\n", err)
		http.Error(w, "Error marshalling patched pod", http.StatusInternalServerError)
		return
	}

	patches, err := jsonpatch.CreatePatch(ar.Request.Object.Raw, patchedPodBytes)
	if err != nil {
		klog.Errorf("==> err: %v\n", err)
		http.Error(w, "Error creating JSON patch", http.StatusInternalServerError)
		return
	}
	klog.Infof("patches: %#v", patches)
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		klog.Errorf("==> err: %v\n", err)
		http.Error(w, "Error marshalling JSON patch", http.StatusInternalServerError)
		return
	}
	pt := admissionv1.PatchTypeJSONPatch
	resp := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:       ar.Request.UID,
			Allowed:   true,
			PatchType: &pt,
			Patch:     patchBytes,
		},
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		klog.Errorf("==>resp json err: %v\n", err)
		http.Error(w, "Error ...", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
	klog.Infof("==> resp: %s\n", resp)

}

func addInit() []map[string]interface{} {
	initContainer := corev1.Container{
		Name:  "init-demo",
		Image: "busybox",
		Command: []string{
			"/bin/sh",
			"-c",
			"echo 'Hello from init container!'",
		},
	}

	patch := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/initContainers",
			"value": []corev1.Container{initContainer},
		},
	}
	return patch

}
