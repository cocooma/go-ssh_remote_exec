package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	// "io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func ssh_do(server *api.AgentMember, user string, cmd string) {
	// func ssh_do(server *api.AgentMember, user string, private ssh.Signer, cmd string) {
	//fmt.Printf("%s (%s) %d\n", server.Name, server.Addr, server.Status)
	// An SSH client is represented with a ClientConn. Currently only
	// the "password" authentication method is supported.
	//
	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig.
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.PublicKeys(private),
			// ssh.Password("password"),
			SSHAgent(),
		},
	}
	client, err := ssh.Dial("tcp", server.Addr+":22", config)
	if err != nil {
		fmt.Printf("Failed to dial %s (%s)\n", server.Name, err)
		return
	}
	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session %s (%s)\n", server.Name, err)
	}
	defer session.Close()
	output, _ := session.Output(cmd)
	fmt.Printf("%s %s", server.Name, string(output))
	defer wg.Done()
}

func scp_do(server *api.AgentMember, user string, file string) {
	// func scp_do(server *api.AgentMember, user string, private ssh.Signer, file string) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.PublicKeys(private),
			// ssh.Password("password"),
			SSHAgent(),
		},
	}
	client, err := ssh.Dial("tcp", server.Addr+":22", config)
	if err != nil {
		fmt.Printf("Failed to dial %s (%s)\n", server.Name, err)
		return
	}
	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session %s (%s)\n", server.Name, err)
	}
	defer session.Close()
	//output, _ := session.Output(cmd)
	//fmt.Printf("%s %s", server.Name, string(output))
	f, err := os.Open(file)
	err = scp.CopyPath(f.Name(), f.Name(), session)
	fmt.Printf("%s\n", err)
	/*if _, err := os.Stat(dest); os.IsNotExist(err) {
	 fmt.Printf("no such file or directory: %s\n", dest)
	} else {
	 fmt.Println("success")
	}*/
	defer wg.Done()
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
	var Include = flag.String("include", "", "Include by pattern")
	var Exclude = flag.String("exclude", "", "Exclude by pattern")
	var members []*api.AgentMember
	flag.Parse()
	// Initialize Consul Client
	consul_client, err := api.NewClient(&api.Config{Address: *Server + ":" + *Port})
	if err != nil {
		panic(err)
	}
	// Grab private key for ssh authentication
	// privateBytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa_exec")
	// if err != nil {
	//   panic("Failed to load private key")
	// }
	// private, err := ssh.ParsePrivateKey(privateBytes)
	// if err != nil {
	//   panic("Failed to parse private key")
	// }
	if *Show {
		members = Listmembers(consul_client)
		Showmembers(members)
		os.Exit(0)
	}
	if *Cmd != "" {
		members = Listmembers(consul_client)
		for _, server := range members {
			if (strings.Contains(server.Name, *Include) || !strings.Contains(server.Name, *Exclude)) && (server.Status == 1) {
				wg.Add(1)
				// go ssh_do(server, *Sshuser, private, *Cmd)
				go ssh_do(server, *Sshuser, *Cmd)
			}
		}
	}
	if *Copy != "" {
		members = Listmembers(consul_client)
		for _, server := range members {
			if (strings.Contains(server.Name, *Include) || !strings.Contains(server.Name, *Exclude)) && (server.Status == 1) {
				wg.Add(1)
				// go scp_do(server, *Sshuser, private, *Copy)
				go scp_do(server, *Sshuser, *Copy)
			}
		}
	}
	wg.Wait()
}
