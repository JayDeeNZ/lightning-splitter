package lnd

import (
	"context"
	"fmt"
	"io/ioutil"
	"lightning-splitter/config"
	"log"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
)

type Client struct {
	client lnrpc.LightningClient
}

func New() *Client {
	return &Client{}
}

func (c *Client) Connect(ctx context.Context) {
	tlsCreds, err := credentials.NewClientTLSFromFile(config.Config.TLSCertPath, "")
	if err != nil {
		log.Println("Unable to generate TLS credentials", err)
		return
	}

	macaroonBytes, err := ioutil.ReadFile(config.Config.MacaroonPath)
	if err != nil {
		log.Println("Unable to read macaroon file", err)
		return
	}

	macaroon := &macaroon.Macaroon{}
	if err = macaroon.UnmarshalBinary(macaroonBytes); err != nil {
		log.Println("Unable to unmarshal macaroon", err)
		return
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(macaroon)),
	}

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.Config.Host, config.Config.Port), opts...)
	if err != nil {
		log.Println("Unable to dial lnd node", err)
		return
	}

	c.client = lnrpc.NewLightningClient(conn)
}

func (c *Client) PrintInfo(ctx context.Context) {
	info, err := c.client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		log.Println("Unable to get info from lnd node:", err)
		return
	}

	log.Printf("Node info: %s [%s]", info.Alias, info.Version)
}
