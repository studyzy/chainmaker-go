for (( i=0; i<10000; i++)) ; do docker-compose -f docker-compose.yaml down; docker-compose -f docker-compose.yaml up -d; sleep $(($RANDOM%100)); done
