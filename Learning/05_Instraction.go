package Learning

//启动redis指令
//redis-server
//或者 redis-cli -h 127.0.0.1 -p 6379
//如果出现/452这种连续的乱码,说明出现了中文,想要查看的花,在启动redis时加上--raw的后缀
//redis-cli -h 127.0.0.1 -p 6379 --raw

//启动consul的指令
//consul agent -dev

//启动mysql指令
//service mysql start	或者	systemctl start mysql

//登录mysql
//mysql -u username -p
//sudo mysql 则是登录root用户root的密码为空

//给用户授予某张表的全部权限
//grant all privileges on search_house.* to 'neko'@'localhost';
