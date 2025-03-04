package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ory/dockertest/v3/docker"

	"github.com/osmosis-labs/osmosis/v7/tests/e2e/chain"
)

func (s *IntegrationTestSuite) connectIBCChains() {
	s.T().Logf("connecting %s and %s chains via IBC", s.chains[0].ChainMeta.Id, s.chains[1].ChainMeta.Id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.hermesResource.Container.ID,
		User:         "root",
		Cmd: []string{
			"hermes",
			"create",
			"channel",
			s.chains[0].ChainMeta.Id,
			s.chains[1].ChainMeta.Id,
			"--port-a=transfer",
			"--port-b=transfer",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(
		err,
		"failed connect chains; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.Require().Containsf(
		errBuf.String(),
		"successfully opened init channel",
		"failed to connect chains via IBC: %s", errBuf.String(),
	)

	s.T().Logf("connected %s and %s chains via IBC", s.chains[0].ChainMeta.Id, s.chains[1].ChainMeta.Id)
}

func (s *IntegrationTestSuite) sendIBC(srcChainID, dstChainID, recipient string, token sdk.Coin) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("sending %s from %s to %s (%s)", token, srcChainID, dstChainID, recipient)

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.hermesResource.Container.ID,
		User:         "root",
		Cmd: []string{
			"hermes",
			"tx",
			"raw",
			"ft-transfer",
			dstChainID,
			srcChainID,
			"transfer",  // source chain port ID
			"channel-0", // since only one connection/channel exists, assume 0
			token.Amount.String(),
			fmt.Sprintf("--denom=%s", token.Denom),
			fmt.Sprintf("--receiver=%s", recipient),
			"--timeout-height-offset=1000",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(
		err,
		"failed to send IBC tokens; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.T().Log("successfully sent IBC tokens")
}

func (s *IntegrationTestSuite) submitProposal(c *chain.Chain) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("submitting upgrade proposal for chain-id: %s", c.ChainMeta.Id)
	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.valResources[c.ChainMeta.Id][0].Container.ID,
		User:         "root",
		Cmd: []string{
			"osmosisd", "tx", "gov", "submit-proposal", "software-upgrade", "v8", "--title=\"v8 upgrade\"", "--description=\"v8 upgrade proposal\"", "--upgrade-height=75", "--upgrade-info=\"\"", fmt.Sprintf("--chain-id=%s", c.ChainMeta.Id), "--from=val", "-b=block", "--yes", "--keyring-backend=test", "--log_format=json",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})

	s.Require().NoErrorf(
		err,
		"failed to submit proposal; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.Require().Truef(
		strings.Contains(outBuf.String(), "code: 0"),
		"tx returned non code 0",
	)

	s.T().Log("successfully submitted proposal")
}

func (s *IntegrationTestSuite) depositProposal(c *chain.Chain) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("depositing to upgrade proposal for chain-id: %s", c.ChainMeta.Id)
	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.valResources[c.ChainMeta.Id][0].Container.ID,
		User:         "root",
		Cmd: []string{
			"osmosisd", "tx", "gov", "deposit", "1", "10000000stake", "--from=val", fmt.Sprintf("--chain-id=%s", c.ChainMeta.Id), "-b=block", "--yes", "--keyring-backend=test",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})

	s.Require().NoErrorf(
		err,
		"failed to deposit to upgrade proposal; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	s.Require().Truef(
		strings.Contains(outBuf.String(), "code: 0"),
		"tx returned non code 0",
	)

	s.T().Log("successfully deposited to proposal")

}

func (s *IntegrationTestSuite) voteProposal(c *chain.Chain) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("voting for upgrade proposal for chain-id: %s", c.ChainMeta.Id)
	for i := range c.Validators {
		exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
			Context:      ctx,
			AttachStdout: true,
			AttachStderr: true,
			Container:    s.valResources[c.ChainMeta.Id][i].Container.ID,
			User:         "root",
			Cmd: []string{
				"osmosisd", "tx", "gov", "vote", "1", "yes", "--from=val", fmt.Sprintf("--chain-id=%s", c.ChainMeta.Id), "-b=block", "--yes", "--keyring-backend=test",
			},
		})
		s.Require().NoError(err)

		var (
			outBuf bytes.Buffer
			errBuf bytes.Buffer
		)

		err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
			Context:      ctx,
			Detach:       false,
			OutputStream: &outBuf,
			ErrorStream:  &errBuf,
		})

		s.Require().NoErrorf(
			err,
			"failed to vote for proposal; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
		)

		s.Require().Truef(
			strings.Contains(outBuf.String(), "code: 0"),
			"tx returned non code 0",
		)

		s.T().Logf("successfully voted for proposal on container: %s", s.valResources[c.ChainMeta.Id][i].Container.ID)
	}
}

func (s *IntegrationTestSuite) chainStatus(containerId string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    containerId,
		User:         "root",
		Cmd: []string{
			"osmosisd", "status",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})

	s.Require().NoErrorf(
		err,
		"failed to query height; stdout: %s, stderr: %s", outBuf.String(), errBuf.String(),
	)

	errBufByte := errBuf.Bytes()
	return errBufByte

}
