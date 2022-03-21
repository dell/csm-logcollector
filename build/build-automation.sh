export PATH=$PATH:/usr/local/go/bin
rm -rf "$WORKSPACE/build"
mkdir "$WORKSPACE/build"
go mod tidy
# application build version
version="v1.0.0.${BUILD_NUMBER}"
cp config.yml "$WORKSPACE/build/."
cp README.md "$WORKSPACE/build/."
go build -o "$WORKSPACE/build/csm-logcollector" -ldflags "-X 'main.version=${version}'"
# scp config and docker files to build agent's Workspace
sshpass -p "dangerous" scp -o StrictHostKeyChecking=no root@10.247.66.65:/home/akash/golang_app/Dockerfile $WORKSPACE
sshpass -p "dangerous" scp -o StrictHostKeyChecking=no root@10.247.66.65:/home/akash/golang_app/build/config.yml $WORKSPACE
images=$(docker images -a | grep "csm-logcollector" | awk '{print $3}')
if [ -z != $images ]
then
docker rmi $images -f
fi
docker build -t csm-logcollector .
# Quay docker registry
docker tag csm-logcollector quay.io/arindam_datta/csm-logcollector:"${version}"
docker tag csm-logcollector quay.io/arindam_datta/csm-logcollector
docker push quay.io/arindam_datta/csm-logcollector
docker push quay.io/arindam_datta/csm-logcollector:"${version}"
docker pull quay.io/arindam_datta/csm-logcollector:"${version}"
# dellemc docker registry
docker tag csm-logcollector dellemc/csm-log-collector:"${version}"
docker push dellemc/csm-log-collector:"${version}"
