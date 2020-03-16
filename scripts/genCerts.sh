#!/bin/bash
usage () {
 echo "Usage: $0 [-o output_directory] [-i IP] [-d DNS] [-u UUID]"
 exit
}

DIRECTORY=$(cd "$(dirname "$0")" && pwd)
DNS="mydomain.adam"
UUID=$(uuidgen)
IP=192.168.0.1

while getopts 'ho:i:d:u:' c
do
 case $c in
  o) DIRECTORY=$OPTARG
     echo "Use with output_directory $DIRECTORY" ;;
  d) DNS=$OPTARG
     echo "Use with DNS $DNS" ;;
  u) UUID=$OPTARG
     echo "Use with UUID $UUID" ;;
  i) IP=$OPTARG
     echo "Use with IP $IP" ;;
  h) usage ;;
  *) usage ;;
 esac
done
test -f /etc/ssl/openssl.cnf || exit 1
cd $DIRECTORY ||exit 1
openssl genrsa -out root-certificate.key 4096
openssl req -x509 -new -nodes -key root-certificate.key -sha256 -subj "/C=RU/ST=SPB/O=MyOrg,Inc./CN=ca" -days 365 -out root-certificate.pem ||exit 1
openssl ecparam -name prime256v1 -genkey -out server-key.pem ||exit 1
openssl req -new -sha256 -key server-key.pem -subj "/C=RU/ST=SPB/O=MyOrg,Inc./CN=$DNS" -reqexts SAN -config <(cat /etc/ssl/openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:$DNS,IP:$IP")) -out server.csr ||exit 1
openssl x509 -req -extfile <(printf "subjectAltName=DNS:$DNS,IP:$IP") -days 365 -in server.csr -CA root-certificate.pem -CAkey root-certificate.key -CAcreateserial -out server.pem ||exit 1
openssl ecparam -name prime256v1 -genkey -out onboard.key.pem ||exit 1
openssl req -new -sha256 -key onboard.key.pem -subj "/C=RU/ST=SPB/O=MyOrg,Inc./CN=$UUID" -out onboard.cert.pem.csr ||exit 1
openssl x509 -req -in onboard.cert.pem.csr -CA root-certificate.pem -CAkey root-certificate.key -CAcreateserial -out onboard.cert.pem -days 365 -sha256 ||exit 1
