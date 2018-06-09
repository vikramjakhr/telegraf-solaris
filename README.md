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
##### Step 1: Download the [latest release tar](https://github.com/vikramjakhr/telegraf-solaris/releases/latest). Example command below.
```
wget https://github.com/vikramjakhr/telegraf-solaris/releases/download/v1.0.0/telegraf-solaris.tar.gz
```

##### Step 2: Extract the tar downloaded in step 1 in /tmp directory by executing below command. It will create directory namely telegraf-solaris in /tmp. 
```
tar -zxvf telegraf-solaris.tar.gz -C /tmp
```
##### Step 3: Create user and group having the name telegraf
```
useradd telegraf
```
##### Step 4: Now copy the /tmp/telegraf-solaris/opt/csw directory to your solaris server's /opt directory:
``` 
cp -r /tmp/telegraf-solaris/opt/csw /opt
```  
##### Step 5: Copy telegraf binary and give it execute permission
```
cp /tmp/telegraf-solaris/bin/telgraf /usr/bin/telgraf; chmod +x /usr/bin/telgraf
```
##### Step 6: Copy telegraf init.d start/stop script:
```
cp /tmp/telegraf-solaris/script/telegraf /etc/init.d/telegraf ; chmod +x /etc/init.d/telegraf
```  
##### Step 7: Create telegraf conf directory /etc/telegraf and telegraf conf file
```
mkdir /etc/telegraf
cp /tmp/telegraf-solaris/conf/telegraf.conf /etc/telegraf/telegraf.conf
```  
##### Step 8: Start telegraf service
```
/etc/init.d/telegraf start
```  
##### Step 9: Verify if service is running or not
```
/etc/init.d/telegraf status
```  
##### Step 10: To enable service in every boot
```
ln -s /etc/init.d/telegraf /etc/rc3.d/S93telegraf
```  

# Building the telegraf binary from source
##### Step 1: Clone the telegraf solaris repository
```
git clone git@github.com:vikramjakhr/telegraf-solaris.git
```
##### Step 2: Execute below command to create a binary named telegraf (Step 4 above is needed for this)
```
cd <path-to-telegraf-solaris-repo>
/opt/csw/bin/gccgo -o telegraf *.go
```
It will create binary in the same directory with the name telegraf. You can now use this binary along with installation steps mentioned above.
