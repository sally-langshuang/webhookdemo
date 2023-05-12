manifest:
	make -C ./certs cert
	make -C ./certs caBundle

RELEASE_NAME ?= demo-w

update: delete install

delete:
	make -C ./certs delete
	helm delete $(RELEASE_NAME)

install:
	make -C ./certs c
	cd ./cmd/demo && go build . && docker build . -t harbor.yusur.tech/yusur_cni/demo:latest
	docker push harbor.yusur.tech/yusur_cni/demo:latest
	crictl pull harbor.yusur.tech/yusur_cni/demo:latest
	helm install $(RELEASE_NAME) ./charts/demo

