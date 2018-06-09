# telegraf-solaris

Telegraf is an extremely powerful and one of the fastest adopted agents in today’s world for collecting,
processing and aggregating different metrics. It’s an extremely versatile agent with a minimal memory footprint
which works on plugin based architecture. It’s open source and written in Go Language. This allows
developers to continuously improve and extend its adaptability across diverse Operating systems and H/Ws.

More information can be referred in links below:
1. https://github.com/influxdata/telegraf

Telegraf is not off-the-shelf supported on Solaris 10 and 11 operating systems. We would need to compile telegraf​ agents from customized source-code for above mentioned operating systems.

# Prerequisites
For the purpose of compilation, we would need gccgo (>=5.5.0) installed on the server with few other libraries.

# Installation Procedure of Telegraf Agent
Step 1. Copy telegraf binary, init script (telegraf) and telegraf.conf and Installed packages.
```
scp -r /tmp/APM/<OS-version>/telegraf/​ <destination>​:/tmp/
```
Step 2. Execute below steps on <destination> ​Solaris server:
```
Create user and group telegraf
```
Step 3. Extract package tar /tmp/telegraf/opt-csw.tar:
``` 
cd / ; tar -xvf /tmp/telegraf/opt-csw.tar OR​ tar -xvf /tmp/telegraf/opt-csw.tar -C /
```  
Step 4. Copy telegraf binary and give it execute permission
```
cp /tmp/telegraf/bin/telgraf /usr/bin/telgraf; chmod +x /usr/bin/telgraf
```
Step 5. Copy telegraf init.d start/stop script:
```
cp /tmp/telegraf/script/telegraf /etc/init.d/telegraf ; chmod +x /etc/init.d/telegraf
```  
Step 6. Create telegraf conf directory /etc/telegraf and telegraf conf
```
mkdir /etc/telegraf
cp /tmp/telegraf/conf/telegraf.conf /etc/telegraf/telegraf.conf
```  
Step 7. Start telegraf service
```
/etc/init.d/telegraf start
```  
Step 8. Verify if service is running or not
```
/etc/init.d/telegraf status
```  
Step 9. To enable service in every boot
```
ln -s /etc/init.d/telegraf /etc/rc3.d/S93telegraf
```  
