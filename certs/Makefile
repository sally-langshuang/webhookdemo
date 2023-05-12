NAME ?= demo
NAMESPACE ?= default
EXPIRE_DAY ?= 10000
DNS_FILE = openssl-san.cnf
SECRET_NAME ?= demo-secret

manifest: create-dns create-cert caBundle

create-dns:
	sed 's/\$(svcDns)/$(NAME).$(NAMESPACE).svc/g' > $(DNS_FILE)

create-cert:
	# 生成CA私钥 ca.key
	openssl genrsa -out ca.key 2048
	# 生成CA证书 ca.crt
	openssl req -x509 -new -nodes -key ca.key -subj "/CN=$(NAME)-ca" -days $(EXPIRE_DAY) -out ca.crt
	# 生成webhook服务私钥 webhook.key
	openssl genrsa -out webhook.key 2048
	# 创建证书签名请求 webhook.csr
	openssl req -new -key webhook.key -subj "/CN=$(NAME).$(NAMESPACE).svc" -out webhook.csr -config  $(DNS_FILE)
	# 生成证书 webhook.crt
	openssl x509 -req -in webhook.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out webhook.crt -days $(EXPIRE_DAY) -extensions req_ext -extfile openssl-san.cnf
	# 复制ca.crt到helm目录
	cp ca.crt ../charts/webhook/templates
	# 查看证书DNS等
	openssl x509 -in webhook.crt -text -noout


caBundle:
	CA_BUNDLE=$(cat ca.crt | base64 | tr -d '\n')
	echo $CA_BUNDLE > ../charts/demo/templates/caBundle
	echo success

create-secret:
	kubectl create secret tls $(SECRET_NAME) --key webhook.key --cert webhook.crt -n $(NAMESPACE)

delete-secret:
	kubectl delete secret $(SECRET_NAME) -n $(NAMESPACE)




