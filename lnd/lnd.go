package lnd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"lightning-splitter/config"

	log "github.com/sirupsen/logrus"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/lightningnetwork/lnd/record"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
)

type Client struct {
	client       lnrpc.LightningClient
	routerClient routerrpc.RouterClient
}

const (
	paymentTimeout int32 = 60
	keysendAmount  int64 = 100
)

func New() *Client {
	return &Client{}
}

func (c *Client) Connect(ctx context.Context) {
	tlsCreds, err := credentials.NewClientTLSFromFile(config.Config.TLSCertPath, "")
	if err != nil {
		log.WithError(err).Fatal("Unable to generate TLS credentials")
		return
	}

	macaroonBytes, err := ioutil.ReadFile(config.Config.MacaroonPath)
	if err != nil {
		log.WithError(err).Fatal("Unable to read macaroon file")
		return
	}

	macaroon := &macaroon.Macaroon{}
	if err = macaroon.UnmarshalBinary(macaroonBytes); err != nil {
		log.WithError(err).Fatal("Unable to unmarshal macaroon")
		return
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(macaroon)),
	}

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.Config.Host, config.Config.Port), opts...)
	if err != nil {
		log.WithError(err).Fatal("Unable to dial lnd node")
		return
	}

	c.client = lnrpc.NewLightningClient(conn)
	c.routerClient = routerrpc.NewRouterClient(conn)

	log.Infof("Successfully connected to lnd node %s:%d", config.Config.Host, config.Config.Port)
}

func (c *Client) GetNodeInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return c.client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}

func (c *Client) PrintInfo(ctx context.Context) {
	info, err := c.GetNodeInfo(ctx)
	if err != nil {
		log.WithError(err).Error("Unable to get info from lnd node")
		return
	}

	log.Infof("Node info: %s [%s]", info.Alias, info.Version)
}

func (c *Client) SubscribeToInvoiceEvents(ctx context.Context) {
	subscribeInvoicesClient, err := c.client.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{})
	if err != nil {
		log.WithError(err).Fatal("Unable to subscribe to invoice events")
	}

	log.Info("Listening for invoice events...")

	for {
		invoice, err := subscribeInvoicesClient.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.WithError(err).Fatal("An error occurred while processing invoice events")
		}

		switch invoice.State {
		case lnrpc.Invoice_OPEN:
			log.Info("Invoice created")
		case lnrpc.Invoice_SETTLED:
			log.Infof("Invoice settled: %d satoshis", invoice.AmtPaidSat)
		}

		log.Infof("%x", invoice.PaymentAddr)
		log.Infof("%s", invoice.PaymentRequest)

		paymentRequest, err := c.client.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: invoice.PaymentRequest})
		if err != nil {
			log.WithError(err).Error("Unable to decode payment request")
			continue
		}

		log.Infof("Node public key: %s", paymentRequest.Destination)
	}
}

func (c *Client) RegisterPayee(ctx context.Context, lnUrl string) error {
	paymentRequest, err := c.client.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: lnUrl})
	if err != nil {
		log.WithError(err).Error("Unable to decode payment request")
		return err
	}

	log.Infof("Registering node with public key: %s", paymentRequest.Destination)

	doPayInvoice := false
	if doPayInvoice {
		// Check route exists by fulfilling payment
		invoicePayment := routerrpc.SendPaymentRequest{
			PaymentRequest:    lnUrl,
			TimeoutSeconds:    paymentTimeout,
			NoInflightUpdates: true,
		}

		if err = c.sendPayment(ctx, invoicePayment); err != nil {
			log.WithError(err).Error("An error occurred trying to pay invoice")
			return err
		}

		log.Infof("Invoice paid successfully!")
	}

	// Try 'keysend'
	pubkey, err := hex.DecodeString(paymentRequest.Destination)
	if err != nil {
		log.WithError(err).Error("Unable to decode node ID to hex bytes")
		return err
	}

	var preimage lntypes.Preimage
	if _, err := rand.Read(preimage[:]); err != nil {
		log.WithError(err).Error("Unable to generate preimage")
		return err
	}
	hash := preimage.Hash()

	var keysendRequest = routerrpc.SendPaymentRequest{
		PaymentHash:       hash[:],
		Dest:              pubkey,
		Amt:               keysendAmount,
		TimeoutSeconds:    paymentTimeout,
		NoInflightUpdates: true,
		DestCustomRecords: map[uint64][]byte{
			record.KeySendType: preimage[:],
		},
		FinalCltvDelta: 40,
	}

	if err = c.sendPayment(ctx, keysendRequest); err != nil {
		log.WithError(err).Error("An error occurred trying to send keysend payment")
		return err
	}

	log.Infof("Keysend payment sent successfully!")
	return nil
}

func (c *Client) sendPayment(ctx context.Context, paymentRequest routerrpc.SendPaymentRequest) error {
	stream, err := c.routerClient.SendPaymentV2(ctx, &paymentRequest)
	if err != nil {
		return err
	}

	update, err := stream.Recv()
	if err != nil {
		return err
	}

	if update.Status != lnrpc.Payment_SUCCEEDED {
		return fmt.Errorf("payment failed: %v", update.FailureReason)
	}

	return nil
}
