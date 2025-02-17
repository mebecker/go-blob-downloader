# Demonstration of using AKS Workload Identity to access azure storage using golang

## Caveats

The golang code is messy - it's just a bunch of copilot generated code to show that things work. Obviously not production ready at all.

## Docs

Just in case - [What is AKS Workload Identity?](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview?tabs=dotnet)

[Walkthrough for setting up AKS Workload Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster)

## Steps to set up

```bash
export ACR_NAME=###YOUR VALUE HERE###
export TOKEN_NAME=go-blob-downloader-token
export LOCATION=###YOUR VALUE HERE###
export CLUSTER_NAME=###YOUR VALUE HERE###
export RESOURCE_GROUP=###YOUR VALUE HERE###
export USER_ASSIGNED_IDENTITY_NAME=msi-go-blob-downloader
export SERVICE_ACCOUNT_NAME=go-blob-downloader-sa
export FEDERATED_IDENTITY_CREDENTIAL_NAME=go-blob-downloader-FedIdentity
export NAMESPACE=go-blob-downloader
export ACR_TOKEN=$(az acr token create --name $TOKEN_NAME --registry $ACR_NAME --scope-map _repositories_push_metadata_write --expiration $(date -u -d "+1 day" +"%Y-%m-%dT%H:%M:%SZ") --query "credentials.passwords[0].value" --output tsv)
export AKS_OIDC_ISSUER="$(az aks show --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --query "oidcIssuerProfile.issuerUrl" --output tsv)"

az login
docker login $ACR_NAME.azurecr.io

docker build -t go-blob-downloader .  
docker tag go-blob-downloader $ACR_NAME.azurecr.io/go-blob-downloader:latest
docker push $ACR_NAME.azurecr.io/go-blob-downloader:latest 

export USER_ASSIGNED_CLIENT_ID="$(az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --location "${LOCATION}"  --query clientId --output tsv)"
az identity federated-credential create --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --issuer "${AKS_OIDC_ISSUER}" --subject system:serviceaccount:"${NAMESPACE}":"${SERVICE_ACCOUNT_NAME}" --audience api://AzureADTokenExchange

kubectl create namespace $NAMESPACE

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: "${USER_ASSIGNED_CLIENT_ID}"
  name: "${SERVICE_ACCOUNT_NAME}"
  namespace: ${NAMESPACE}
EOF



kubectl create secret go-blob-downloader-regcred docker-registry --docker-server=$ACR_NAME.azurecr.io --docker-username=$TOKEN_NAME --docker-password=$ACR_TOKEN --save-config --dry-run=client -o json | kubectl apply -f -
kubectl apply -f deployment.yaml

kubectl port-forward -n go-blob-downloader $(kubectl get pods -n go-blob-downloader -o name --no-headers=true) 8080:8080 &

curl http://127.0.0.1:8080/?bloburl=https://{storageaccountname}.blob.core.windows.net/data/file.csv

```
