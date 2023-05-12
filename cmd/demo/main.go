package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gomodules.xyz/jsonpatch/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
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
	// 加载TLS证书
	cert, err := tls.LoadX509KeyPair("webhook.crt", "webhook.key")
	if err != nil {
		log.Fatalf("Failed to load key pair: %v", err)
	}

	server := &http.Server{
		Addr: ":8443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	http.HandleFunc("/mutate", mutateHandler)
	fmt.Println("Webhook server started")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
	server.ListenAndServeTLS("", "")
}

func mutateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("==> mutateHandler")
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	fmt.Printf("==> body: %s\n", body)
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
	if err != nil {
		fmt.Printf("==> err: %v\n", err)
		http.Error(w, "Failed to unmarshal pod", http.StatusBadRequest)
		return
	}

	if s, ok := pod.Annotations["cal"]; ok && strings.EqualFold(s, "1+1") {
		pod.Annotations["res"] = "2"
	}

	patchedPodBytes, err := json.Marshal(pod)
	if err != nil {
		fmt.Printf("==> err: %v\n", err)
		http.Error(w, "Error marshalling patched pod", http.StatusInternalServerError)
		return
	}

	patches, err := jsonpatch.CreatePatch(ar.Request.Object.Raw, patchedPodBytes)
	if err != nil {
		fmt.Printf("==> err: %v\n", err)
		http.Error(w, "Error creating JSON patch", http.StatusInternalServerError)
		return
	}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		fmt.Printf("==> err: %v\n", err)
		http.Error(w, "Error marshalling JSON patch", http.StatusInternalServerError)
		return
	}
	fmt.Printf("==> patchBytes: %v\n", patchBytes)
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
	//initContainer := corev1.Container{
	//	Name:  "init-demo",
	//	Image: "busybox",
	//	Command: []string{
	//		"/bin/sh",
	//		"-c",
	//		"echo 'Hello from init container!'",
	//	},
	//}
	//
	//patch := []map[string]interface{}{
	//	{
	//		"op":    "add",
	//		"path":  "/spec/initContainers",
	//		"value": []corev1.Container{initContainer},
	//	},
	//}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("==>resp json err: %v\n", err)
		http.Error(w, "Error ...", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
	fmt.Println("==> mutateHandler end")

}
