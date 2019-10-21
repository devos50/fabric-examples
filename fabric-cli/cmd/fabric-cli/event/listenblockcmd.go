/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package event

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-examples/fabric-cli/action"
        "github.com/securekey/fabric-examples/fabric-cli/printer"
	cliconfig "github.com/securekey/fabric-examples/fabric-cli/config"
	fabriccmn "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var listenBlockCmd = &cobra.Command{
	Use:   "listenblock",
	Short: "Listen to block events.",
	Long:  "Listen to block events",
	Run: func(cmd *cobra.Command, args []string) {
		action, err := newlistenBlockAction(cmd.Flags())
		if err != nil {
			cliconfig.Config().Logger().Errorf("Error while initializing listenBlockAction: %v", err)
			return
		}

		defer action.Terminate()

		err = action.invoke()
		if err != nil {
			cliconfig.Config().Logger().Errorf("Error while running listenBlockAction: %v", err)
		}
	},
}

func getListenBlockCmd() *cobra.Command {
	flags := listenBlockCmd.Flags()
	cliconfig.InitChannelID(flags)
	cliconfig.InitPeerURL(flags, "", "The URL of the peer on which to listen for events, e.g. localhost:7051")
	cliconfig.InitSeekType(flags)
	cliconfig.InitBlockNum(flags)
	return listenBlockCmd
}

type listenBlockAction struct {
	action.Action
	inputEvent
}

func newlistenBlockAction(flags *pflag.FlagSet) (*listenBlockAction, error) {
	action := &listenBlockAction{inputEvent: inputEvent{done: make(chan bool)}}
	err := action.Initialize(flags)
	return action, err
}

func (a *listenBlockAction) invoke() error {
	eventClient, err := a.EventClient(event.WithBlockEvents(), event.WithSeekType(cliconfig.Config().SeekType()), event.WithBlockNum(cliconfig.Config().BlockNum()))
	if err != nil {
		return err
	}

	//fmt.Printf("Registering block event\n")
	//f, err := os.Create("blocks.txt")

	breg, beventch, err := eventClient.RegisterBlockEvent()
	if err != nil {
		return errors.WithMessage(err, "Error registering for block events")
	}
	defer eventClient.Unregister(breg)

	enterch := a.WaitForEnter()
	for {
		select {
		case _, _ = <-enterch:
			//fmt.Printf("Done, got enter")
			//return nil
		case event, ok := <-beventch:
			if !ok {
				return errors.WithMessage(err, "unexpected closed channel while waiting for block event")
			}

			var txs = 0
			var curTime = time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))

			// determine transactions in block
			block := event.Block
			for i := range block.Data.Data {
				envelope := printer.ExtractEnvelopeOrPanic(block, i)
				payload := printer.ExtractPayloadOrPanic(envelope)

				chdr, err := printer.UnmarshalChannelHeader(payload.Header.ChannelHeader)
				if err != nil {
					panic(err)
				}

				if fabriccmn.HeaderType(chdr.Type) == fabriccmn.HeaderType_ENDORSER_TRANSACTION {
					_, err := printer.GetTransaction(payload.Data)
					if err != nil {
						panic(errors.Errorf("Bad envelope: %v", err))
					}
					txs++
				}
			}

			fmt.Printf("%d", block.Header.Number)
			fmt.Printf(",")
			fmt.Printf("%d", curTime)
			fmt.Printf(",")
			fmt.Printf("%d", txs)
			fmt.Printf("\n")
			//f.Sync()
		}
	}
}
