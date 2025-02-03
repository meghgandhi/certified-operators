#!/bin/bash
		
# Copyright 2024 IBM Corporation
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -x

IDX=0
VAULT_LOGIN_ADDR="VAULT_LOGIN_ADDR_$IDX"
VAULT_SECRET_ADDR="VAULT_SECRET_ADDR_$IDX"
REGISTRY_NAME="REGISTRY_NAME_$IDX"
REGISTRY_USER="REGISTRY_USER_$IDX"
VAULT_ROLE="VAULT_ROLE_$IDX"
CERT_PATH="CERT_PATH_$IDX"
SA_TOKEN="SA_TOKEN_$IDX"
SECRET_KEY="SECRET_KEY_$IDX"
CONFIG_JSON=('{"auths":{}}')

while [[ -v $VAULT_ROLE ]]; do
    ACCOUNT_TOKEN=$(cat ${!SA_TOKEN}/token)
    LOGIN_PAYLOAD=$(jq -n --arg role "${!VAULT_ROLE}" --arg jwt "$ACCOUNT_TOKEN" '$ARGS.named')
    VAULT_TOKEN=$(curl --cacert "${!CERT_PATH}/ca.crt" --data "$LOGIN_PAYLOAD" -X POST ${!VAULT_LOGIN_ADDR} | jq --raw-output '.auth.client_token') 
    VAULT_SECRET=$(curl --cacert "${!CERT_PATH}/ca.crt" --header "X-Vault-Token: $VAULT_TOKEN" ${!VAULT_SECRET_ADDR} | jq -r ".data.data[\"${!SECRET_KEY}\"]")
    DOCKER_AUTH=$(echo -n "${!REGISTRY_USER}:${VAULT_SECRET}" | base64 | tr -d ' '| tr -d '\n')
    CONFIG_JSON[0]=$(echo "${CONFIG_JSON[0]}"| jq --raw-output --arg registry "${!REGISTRY_NAME}" --arg auth "${DOCKER_AUTH}" --arg user "${!REGISTRY_USER}" --arg token "${VAULT_SECRET}" '.auths += {($registry):{auth: $auth, username: $user, password:$token}}')
	
    ((IDX+=1))
	VAULT_LOGIN_ADDR="VAULT_LOGIN_ADDR_$IDX"
	VAULT_SECRET_ADDR="VAULT_SECRET_ADDR_$IDX"
	VAULT_ROLE="VAULT_ROLE_$IDX"
	CERT_PATH="CERT_PATH_$IDX"
	REGISTRY_NAME="REGISTRY_NAME_$IDX"
	REGISTRY_USER="REGISTRY_USER_$IDX"
	SA_TOKEN="SA_TOKEN_$IDX"
	SECRET_KEY="SECRET_KEY_$IDX"
done
echo "${CONFIG_JSON[0]}" > /opt/scanner/vault/auth/tmp.json
jq -s '.[0] * .[1]' /opt/scanner/auth/config.json /opt/scanner/vault/auth/tmp.json > /opt/scanner/vault/auth/config.json
rm /opt/scanner/vault/auth/tmp.json