package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	flag "github.com/docker/docker/pkg/mflag"
	"github.com/hashicorp/consul/api"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	url, port, sshuser, cmd, copy, destfile, include, exclude string
	timeout                                                   int
	show                                                      bool
	wg                                                        sync.WaitGroup
)

func sSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func sshDo(server *api.AgentMember, user string, cmd string, timeout int) {
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
			sSHAgent(),
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

func scpDo(server *api.AgentMember, user string, file string, destfile string, timeout int) {
	// func scpDo(server *api.AgentMember, user string, private ssh.Signer, file string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happend:", e)
		}
	}()

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.PublicKeys(private),
			// ssh.Password("password"),
			sSHAgent(),
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

func listmembers(consulClient *api.Client) []*api.AgentMember {
	status := consulClient.Agent()
	members, _ := status.Members(false)
	return members
}

func showMembers(members []*api.AgentMember) {
	for _, server := range members {
		fmt.Println(server.Name)
	}
}

func main() {
	flag.StringVar(&url, []string{"u", "-url"}, "localhost", "Consul member endpoint. Default: localhost")
	flag.StringVar(&port, []string{"p", "-port"}, "8500", "Consul members endpoint port. Default: 8500")
	flag.StringVar(&sshuser, []string{"us", "-user"}, "", "Cli user name")
	flag.BoolVar(&show, []string{"lcm", "-show-consul-members"}, false, "List consul members")
	flag.StringVar(&cmd, []string{"c", "-cmd"}, "", "Command to run on the server")
	flag.StringVar(&copy, []string{"cp", "-copy"}, "", "File to copy to the server")
	flag.StringVar(&destfile, []string{"df", "-destfile"}, "", "Destination file to copy to the server")
	flag.StringVar(&include, []string{"in", "-include"}, "", "Include by pattern")
	flag.StringVar(&exclude, []string{"ex", "-exclude"}, "", "Exclude by pattern")
	flag.IntVar(&timeout, []string{"t", "-timeout"}, 5, "Connection timeout")
	flag.Parse()
	var members []*api.AgentMember

	// Initialize Consul Client
	consulClient, err := api.NewClient(&api.Config{Address: url + ":" + port})
	if err != nil {
		panic(err)
	}

	if show {
		members = listmembers(consulClient)
		showMembers(members)
		os.Exit(0)
	}
	if cmd != "" {
		members = listmembers(consulClient)
		wg.Add(len(members))
		for _, server := range members {
			if (strings.Contains(server.Name, include) || !strings.Contains(server.Name, exclude)) && (server.Status == 1) {
				go func(server *api.AgentMember) {
					sshDo(server, sshuser, cmd, timeout)
					wg.Done()
				}(server)
			}
		}
	}
	if copy != "" {
		members = listmembers(consulClient)
		wg.Add(len(members))
		for _, server := range members {
			if (strings.Contains(server.Name, include) || !strings.Contains(server.Name, exclude)) && (server.Status == 1) {
				go func(server *api.AgentMember) {
					scpDo(server, sshuser, copy, destfile, timeout)
					wg.Done()
				}(server)
			}
		}
	}
	wg.Wait()

}
