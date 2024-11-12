export PRODUCT="gateway"
export VERSION="latest"
export VERSIONS="latest,v1.3.0"
export BRANCH="main"
export OUTPUT="/tmp/gateway-docs"

cp config.yaml config.bak
# Replace placeholder (e.g., "BRANCH") in config.toml with the VERSION environment variable
sed -i '' "s/BRANCH/$BRANCH/g" config.yaml


hugo.old  --minify --theme book  --destination="${OUTPUT}"/"${PRODUCT}"/"$VERSION"\
                             		--baseURL="/${PRODUCT}"/"$VERSION" 1> /dev/null

# Restore the original config.toml
mv config.bak config.yaml