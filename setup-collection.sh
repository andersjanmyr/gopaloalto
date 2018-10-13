#!/usr/local/bin/bash

set -o errexit

collection_id=gopaloalto

if ! aws rekognition describe-collection --collection-id $collection_id; then
  aws rekognition create-collection --collection-id $collection_id
fi

bucket='gopaloalto-photos'

function index_image() {
  name=$1
  only_name=${name%.*}
  IFS='_' read -ra names <<< "$only_name"
  firstname=${names[0]^}
  lastname=${names[1]^}
  echo "Indexing $firstname $lastname"
  aws rekognition index-faces \
    --image "{\"S3Object\":{\"Bucket\":\"$bucket\",\"Name\":\"$name\"}}" \
    --collection-id "gopaloalto" \
    --max-faces 1 \
    --quality-filter "AUTO" \
    --detection-attributes "ALL" \
    --external-image-id "${firstname}_${lastname}"
}

aws s3 ls s3://$bucket | while read line; do
  filename=$(echo $line | awk '{ print $4 }');
  index_image $filename
done
