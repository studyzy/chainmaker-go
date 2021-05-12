cd ../../config-sql
# 如何使用
# how to use
# sqlite to mysql: sh init_mysql_config.sh
# mysql to sqlite: sh init_mysql_config.sh 1

sql_type=$1


if  [[ ! -n $sql_type ]] ;then
  # to mysql config
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org1/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org2/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org3/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org4/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org5/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org6/chainmaker.yml
  sed -i "s%sqldb_type: sqlite%sqldb_type:  mysql%g" wx-org7/chainmaker.yml

  sed -i "s%dsn: ../data/org1/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3307)/mysql%g" wx-org1/chainmaker.yml
  sed -i "s%dsn: ../data/org2/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3308)/mysql%g" wx-org2/chainmaker.yml
  sed -i "s%dsn: ../data/org3/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3309)/mysql%g" wx-org3/chainmaker.yml
  sed -i "s%dsn: ../data/org4/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3310)/mysql%g" wx-org4/chainmaker.yml
  sed -i "s%dsn: ../data/org5/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3311)/mysql%g" wx-org5/chainmaker.yml
  sed -i "s%dsn: ../data/org6/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3312)/mysql%g" wx-org6/chainmaker.yml
  sed -i "s%dsn: ../data/org7/state.db%dsn:  root:passw0rd@tcp(192.168.1.89:3313)/mysql%g" wx-org7/chainmaker.yml
else
  # to sqlite config
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org1/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org2/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org3/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org4/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org5/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org6/chainmaker.yml
  sed -i "s%sqldb_type:  mysql%sqldb_type: sqlite%g" wx-org7/chainmaker.yml

  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3307)/mysql%dsn: ../data/org1/state.db%g" wx-org1/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3308)/mysql%dsn: ../data/org2/state.db%g" wx-org2/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3309)/mysql%dsn: ../data/org3/state.db%g" wx-org3/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3310)/mysql%dsn: ../data/org4/state.db%g" wx-org4/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3311)/mysql%dsn: ../data/org5/state.db%g" wx-org5/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3312)/mysql%dsn: ../data/org6/state.db%g" wx-org6/chainmaker.yml
  sed -i "s%dsn:  root:passw0rd@tcp(192.168.1.89:3313)/mysql%dsn: ../data/org7/state.db%g" wx-org7/chainmaker.yml
fi
