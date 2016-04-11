## Remote ssh exec and scp tool ##

It queries the consul api in order to retrieve the available nodes.

### Usage
Remote exec

```
ssh_remote_exec -server 'consul node ip' -user 'user.name' -cmd 'uptime'

```
SCP

```
ssh_remote_exec  -server 'consul node ip' -user 'user.name' -copy 'source file' -destfile 'destination file'
```

#### To Do
- Enable password auth. Currently the tool only supports agent based auth.

