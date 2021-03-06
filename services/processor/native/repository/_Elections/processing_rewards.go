// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"sort"
)

/***
 * Rewards.
 * Rewards constants are annual!!
 */

var ELECTION_PARTICIPATION_MAX_REWARD = uint64(60000000)
var ELECTION_PARTICIPATION_MAX_STAKE_REWARD_PERCENT = uint64(8)
var ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD = uint64(40000000)
var ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT = uint64(10)
var ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER = 10
var ELECTION_VALIDATOR_INTRODUCTION_REWARD = uint64(1000000)
var ELECTION_VALIDATOR_MAX_STAKE_REWARD_PERCENT = uint64(4)
var ANNUAL_TO_ELECTION_FACTOR = uint64(11723)

func _processRewards(totalVotes uint64, elected [][20]byte, participants [][20]byte, participantStakes map[[20]byte]uint64, guardiansAccumulatedStake map[[20]byte]uint64) {
	_processRewardsParticipants(totalVotes, participants, participantStakes)
	_processRewardsGuardians(totalVotes, guardiansAccumulatedStake)
	_processRewardsValidators(elected)
}

func _processRewardsParticipants(totalVotes uint64, participants [][20]byte, participantStakes map[[20]byte]uint64) {
	totalReward := _maxRewardForGroup(ELECTION_PARTICIPATION_MAX_REWARD, totalVotes, ELECTION_PARTICIPATION_MAX_STAKE_REWARD_PERCENT)
	fmt.Printf("elections %10d rewards: %d participants total reward is %d \n", getCurrentElectionBlockNumber(), len(participantStakes), totalReward)
	for _, participant := range participants {
		stake := participantStakes[participant]
		reward := safeuint64.Div(safeuint64.Mul(stake, totalReward), totalVotes)
		fmt.Printf("elections %10d rewards: participant %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), participant, stake, reward)
		_addCumulativeParticipationReward(participant[:], reward)
	}
}

func _processRewardsGuardians(totalVotes uint64, guardiansAccumulatedStake map[[20]byte]uint64) {
	fmt.Printf("elections %10d rewards: there are %d guardians with total reward is %d - choosing %d top guardians\n",
		getCurrentElectionBlockNumber(), len(guardiansAccumulatedStake), totalVotes, ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER)
	topGuardians, totalTopVotes := _getTopGuardians(guardiansAccumulatedStake)
	fmt.Printf("elections %10d rewards: top %d guardians with total vote is now %d \n", getCurrentElectionBlockNumber(), len(topGuardians), totalTopVotes)

	_setExcellenceProgramGuardians(topGuardians)
	totalReward := _maxRewardForGroup(ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD, totalTopVotes, ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT)
	fmt.Printf("elections %10d rewards: guardians total reward is %d \n", getCurrentElectionBlockNumber(), totalReward)
	for _, guardian := range topGuardians {
		reward := safeuint64.Div(safeuint64.Mul(guardian.vote, totalReward), totalTopVotes)
		fmt.Printf("elections %10d rewards: guardian %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), guardian.address, guardian.vote, reward)
		_addCumulativeGuardianExcellenceReward(guardian.address[:], reward)
	}
}

func _processRewardsValidators(elected [][20]byte) {
	electionValidatorIntroduction := safeuint64.Div(safeuint64.Mul(ELECTION_VALIDATOR_INTRODUCTION_REWARD, 100), ANNUAL_TO_ELECTION_FACTOR)
	validatorsStake := _getValidatorsStake()
	fmt.Printf("elections %10d rewards: %d validadator introduction reward %d\n", getCurrentElectionBlockNumber(), len(validatorsStake), electionValidatorIntroduction)
	for _, elected := range elected {
		stake := validatorsStake[elected]
		reward := safeuint64.Add(electionValidatorIntroduction, safeuint64.Div(safeuint64.Mul(stake, ELECTION_VALIDATOR_MAX_STAKE_REWARD_PERCENT), ANNUAL_TO_ELECTION_FACTOR))
		fmt.Printf("elections %10d rewards: validator %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), elected, stake, reward)
		_addCumulativeValidatorReward(elected[:], reward)
	}
}

func _getValidatorsStake() (validatorsStake map[[20]byte]uint64) {
	numOfValidators := _getNumberOfValidators()
	validatorsStake = make(map[[20]byte]uint64, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		validator := _getValidatorEthereumAddressAtIndex(i)
		stake := getValidatorStake(validator[:])
		validatorsStake[validator] = stake
		fmt.Printf("elections %10d rewards: validator %x, stake %d\n", getCurrentElectionBlockNumber(), validator, stake)
	}
	return
}

func _maxRewardForGroup(upperMaximum, totalVotes, percent uint64) uint64 {
	upperMaximumPerElection := safeuint64.Div(safeuint64.Mul(upperMaximum, 100), ANNUAL_TO_ELECTION_FACTOR)
	calcMaximumPerElection := safeuint64.Div(safeuint64.Mul(totalVotes, percent), ANNUAL_TO_ELECTION_FACTOR)
	fmt.Printf("elections %10d rewards: uppperMax %d vs. %d = totalVotes %d * percent %d / number of annual election \n", getCurrentElectionBlockNumber(), upperMaximumPerElection, calcMaximumPerElection, totalVotes, percent)
	if calcMaximumPerElection < upperMaximumPerElection {
		return calcMaximumPerElection
	}
	return upperMaximumPerElection
}

func _formatCumulativeParticipationReward(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Participant_CumReward_%s", hex.EncodeToString(delegator)))
}

func getCumulativeParticipationReward(delegator []byte) uint64 {
	return state.ReadUint64(_formatCumulativeParticipationReward(delegator))
}

func _addCumulativeParticipationReward(delegator []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeParticipationReward(delegator), reward)
}

func _formatCumulativeGuardianExcellenceReward(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_CumReward_%s", hex.EncodeToString(guardian)))
}

func getCumulativeGuardianExcellenceReward(guardian []byte) uint64 {
	return state.ReadUint64(_formatCumulativeGuardianExcellenceReward(guardian))
}

func _addCumulativeGuardianExcellenceReward(guardian []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeGuardianExcellenceReward(guardian), reward)
}

func _formatCumulativeValidatorReward(validator []byte) []byte {
	return []byte(fmt.Sprintf("Vaidator_CumReward_%s", hex.EncodeToString(validator)))
}

func getCumulativeValidatorReward(validator []byte) uint64 {
	return state.ReadUint64(_formatCumulativeValidatorReward(validator))
}

func _addCumulativeValidatorReward(validator []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeValidatorReward(validator), reward)
}

func _addCumulativeReward(key []byte, reward uint64) {
	sumReward := safeuint64.Add(state.ReadUint64(key), reward)
	state.WriteUint64(key, sumReward)
}

func _formatExcellenceProgramGuardians() []byte {
	return []byte("Excellence_Program_Guardians")
}

func getExcellenceProgramGuardians() []byte {
	return state.ReadBytes(_formatExcellenceProgramGuardians())
}

func _setExcellenceProgramGuardians(guardians guardianArray) {
	guardiansForSave := make([]byte, 0, len(guardians)*20)
	for _, guardian := range guardians {
		guardiansForSave = append(guardiansForSave, guardian.address[:]...)
	}
	state.WriteBytes(_formatExcellenceProgramGuardians(), guardiansForSave)
}

/***
 * Rewards: Sort top guardians using sort.Interface
 */
func _getTopGuardians(guardiansAccumulatedStake map[[20]byte]uint64) (topGuardiansStake guardianArray, totalVotes uint64) {
	totalVotes = uint64(0)

	guardianList := make(guardianArray, 0, len(guardiansAccumulatedStake))
	for guardian, vote := range guardiansAccumulatedStake {
		guardianList = append(guardianList, &guardianVote{guardian, vote})
	}
	sort.Sort(guardianList)

	i := 0
	for i = 0; i < len(guardianList) && i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		fmt.Printf("elections %10d rewards: top guardian %x, has %d votes\n", _getCurrentElectionBlockNumber(), guardianList[i].address, guardianList[i].vote)
		totalVotes = safeuint64.Add(totalVotes, guardianList[i].vote)
	}
	for i = ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i < len(guardianList); i++ {
		if guardianList[i].vote != guardianList[i-1].vote {
			break
		}
		fmt.Printf("elections %10d rewards: top guardian %x, has %d votes\n", _getCurrentElectionBlockNumber(), guardianList[i].address, guardianList[i].vote)
		totalVotes = safeuint64.Add(totalVotes, guardianList[i].vote)
	}
	if i < len(guardianList) {
		return guardianList[0:i], totalVotes
	} else {
		return guardianList, totalVotes
	}
}

type guardianVote struct {
	address [20]byte
	vote    uint64
}
type guardianArray []*guardianVote

func (s guardianArray) Len() int {
	return len(s)
}

func (s guardianArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s guardianArray) Less(i, j int) bool {
	return s[i].vote > s[j].vote || (s[i].vote == s[j].vote && bytes.Compare(s[i].address[:], s[j].address[:]) > 0)
}
