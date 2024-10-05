set -e

npm install -g @bitnami/readme-generator-for-helm --force
readme-generator --values "${PWD}/chart/values.yaml" --readme "${PWD}/chart/README.md" --config "${PWD}/hack/chart-docs/readme-generator-config.json"
