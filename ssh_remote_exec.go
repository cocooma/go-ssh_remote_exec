package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func ssh_do(server *api.AgentMember, user string, cmd string, timeout int) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops: ", e)
		}
	}()

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.PublicKeys(private),
			// ssh.Password("password"),
			SSHAgent(),
		},
	}

	serverandport := server.Addr + ":22"
	_, err := net.DialTimeout("tcp", serverandport, time.Duration(timeout)*time.Second)
	if err != nil {
		panic("Failed DialTimeout: " + err.Error())
	} else {
		client, err := ssh.Dial("tcp", server.Addr+":22", config)
		if err != nil {
			panic("Failed to dial: " + err.Error() + server.Name)
		} else {
			session, err := client.NewSession()
			if err != nil {
				panic("Failed to create session: " + err.Error())
			} else {
				defer session.Close()
				session.Stdout = os.Stdout
				session.Stderr = os.Stderr
				session.Run(cmd)
			}
		}
	}
}

func scp_do(server *api.AgentMember, user string, file string, destfile string, timeout int) {
	// func scp_do(server *api.AgentMember, user string, private ssh.Signer, file string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops:", e)
		}
	}()

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.PublicKeys(private),
			// ssh.Password("password"),
			SSHAgent(),
		},
	}

	serverandport := server.Addr + ":22"
	_, err := net.DialTimeout("tcp", serverandport, time.Duration(timeout)*time.Second)
	if err != nil {
		panic("Failed DialTimeout: " + err.Error())
	} else {
		client, err := ssh.Dial("tcp", server.Addr+":22", config)
		if err != nil {
			panic("Failed to dial: " + server.Name + " Because the following error: " + err.Error())
		} else {
			session, err := client.NewSession()
			if err != nil {
				panic("Failed to create session: " + err.Error())
			} else {
				f, err := os.Open(file)
				if err != nil {
					panic("It looks like file does not exist" + err.Error())
				} else {
					defer session.Close()
					status := scp.CopyPath(f.Name(), destfile, session)
					if status != nil {
						panic("Something is wrong with the source file" + status.Error())
					} else {
						/*if _, err := os.Stat(dest); os.IsNotExist(err) {
						 fmt.Printf("no such file or directory: %s\n", dest)
						} else {
						 fmt.Println("success")
						}*/
						fmt.Println(server.Addr + " Is done")
						return
					}
				}
			}
		}
	}
}

func Listmembers(consul_client *api.Client) []*api.AgentMember {
	status := consul_client.Agent()
	members, _ := status.Members(false)
	return members
}

func Showmembers(members []*api.AgentMember) {
	for _, server := range members {
		fmt.Println(server.Name)
	}
}

func main() {
	var Server = flag.String("server", "localhost", "Server to connect to")
	var Port = flag.String("port", "8500", "Port to connect to")
	var Sshuser = flag.String("user", "", "Username")
	var Show = flag.Bool("show", false, "Show a list of members")
	var Cmd = flag.String("cmd", "", "Command to run on the server")
	var Copy = flag.String("copy", "", "File to copy on the server")
	var destfile = flag.String("destfile", "", "Destination file to copy on the server")
	var Include = flag.String("include", "", "Include by pattern")
	var Exclude = flag.String("exclude", "", "Exclude by pattern")
	var Timeout = flag.Int("timeout", 5, "Connection timeout")
	var members []*api.AgentMember
	flag.Parse()
	// Initialize Consul Client
	consul_client, err := api.NewClient(&api.Config{Address: *Server + ":" + *Port})
	if err != nil {
		panic(err)
	}

	if *Show {
		members = Listmembers(consul_client)
		Showmembers(members)
		os.Exit(0)
	}
	if *Cmd != "" {
		members = Listmembers(consul_client)
		wg.Add(len(members))
		for _, server := range members {
			if (strings.Contains(server.Name, *Include) || !strings.Contains(server.Name, *Exclude)) && (server.Status == 1) {
				go func(server *api.AgentMember) {
					ssh_do(server, *Sshuser, *Cmd, *Timeout)
					wg.Done()
				}(server)
			}
		}
	}
	if *Copy != "" {
		members = Listmembers(consul_client)
		wg.Add(len(members))
		for _, server := range members {
			if (strings.Contains(server.Name, *Include) || !strings.Contains(server.Name, *Exclude)) && (server.Status == 1) {
				go func(server *api.AgentMember) {
					scp_do(server, *Sshuser, *Copy, *destfile, *Timeout)
					wg.Done()
				}(server)
			}
		}
	}
	wg.Wait()

}
