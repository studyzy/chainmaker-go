FILE_PATH=$1

if  [[ ! -n $FILE_PATH ]] ;then
  FILE_PATH="docker-compose1.yml"
fi
docker-compose -f $FILE_PATH down