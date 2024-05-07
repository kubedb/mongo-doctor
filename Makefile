all: image
	kubectl apply -f yamls/rbac.yaml
	kubectl apply -f yamls/job.yaml

run:
	kubectl delete -f yamls/job.yaml || true
	$(MAKE) image
	kubectl apply -f yamls/job.yaml

image:
	docker build -t arnobkumarsaha/mongo-doctor .
	docker push arnobkumarsaha/mongo-doctor
	#kind load docker-image arnobkumarsaha/mongo-doctor

re:
	kubectl delete -f yamls/job.yaml || true
	kubectl apply -f yamls/job.yaml

clean:
	kubectl delete -f yamls/job.yaml
	kubectl delete -f yamls/rbac.yaml