package accesscontrol

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/localconf/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

// *************************************
// 		lookUpPolicyByTxType
// *************************************

func (cp *certACProvider) lookUpPolicyByTxType(txType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByTxType(
		txType, blockVersion,
		cp.acService.latestPolicyMap, cp.acService.txTypePolicyMap)
}

func (pp *permissionedPkACProvider) lookUpPolicyByTxType(txType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByTxType(
		txType, blockVersion,
		pp.acService.latestPolicyMap, pp.acService.txTypePolicyMap)
}

func (pk *pkACProvider) lookUpPolicyByTxType(txType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByTxType(
		txType, blockVersion,
		pk.latestPolicyMap, pk.txTypePolicyMap)
}

// *************************************
// 		lookUpPolicyByTxType
// *************************************

func (cp *certACProvider) lookUpPolicyByMsgType(msgType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByMsgType(
		msgType, blockVersion,
		cp.acService.latestPolicyMap, cp.acService.msgTypePolicyMap)
}

func (pp *permissionedPkACProvider) lookUpPolicyByMsgType(msgType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByMsgType(
		msgType, blockVersion,
		pp.acService.latestPolicyMap, pp.acService.msgTypePolicyMap)
}

func (pk *pkACProvider) lookUpPolicyByMsgType(msgType string, blockVersion uint32) (*policy, error) {
	return lookUpPolicyByMsgType(
		msgType, blockVersion,
		pk.latestPolicyMap, pk.msgTypePolicyMap)
}

// *************************************
// 		findFromSenderPolicies
// *************************************

func (cp *certACProvider) findFromSenderPolicies(resourceName string, blockVersion uint32) (*policy, error) {
	return findFromSenderPolicies(
		resourceName, blockVersion,
		cp.acService.latestPolicyMap, cp.acService.senderPolicyMap)
}

func (pp *permissionedPkACProvider) findFromSenderPolicies(resourceName string, blockVersion uint32) (*policy, error) {
	return findFromSenderPolicies(
		resourceName, blockVersion,
		pp.acService.latestPolicyMap, pp.acService.senderPolicyMap)
}

func (pk *pkACProvider) findFromSenderPolicies(resourceName string, blockVersion uint32) (*policy, error) {
	return findFromSenderPolicies(
		resourceName, blockVersion,
		pk.latestPolicyMap, pk.senderPolicyMap)
}

// *************************************
// 		findFromEndorsementsPolicies
// *************************************

func (cp *certACProvider) findFromEndorsementsPolicies(resourceName string, blockVersion uint32) (*policy, error) {
	return findFromEndorsementsPolicies(
		resourceName, blockVersion,
		cp.acService.latestPolicyMap, cp.acService.resourceNamePolicyMap)
}

func (pp *permissionedPkACProvider) findFromEndorsementsPolicies(
	resourceName string, blockVersion uint32) (*policy, error) {
	return findFromEndorsementsPolicies(
		resourceName, blockVersion,
		pp.acService.latestPolicyMap, pp.acService.resourceNamePolicyMap)
}

func (pk *pkACProvider) findFromEndorsementsPolicies(resourceName string, blockVersion uint32) (*policy, error) {
	return findFromEndorsementsPolicies(
		resourceName, blockVersion,
		pk.latestPolicyMap, pk.resourceNamePolicyMap)
}

// ****************************************************
//  function utils
// ****************************************************

func verifyMsgTypePrincipal(p acProvider,
	principal protocol.Principal, blockVersion uint32) (allow bool, err error) {

	if p.getTotalVoterNum() <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := p.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	pol, err := p.lookUpPolicyByMsgType(principal.GetResourceName(), blockVersion)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return p.verifyPrincipalPolicy(principal, refinedPrincipal, pol)
}

func verifyTxTypePrincipal(p acProvider, tx *commonPb.Transaction,
	txBytes []byte, blockVersion uint32, bypassSignVerify bool) (allow bool, err error) {

	if p.getTotalVoterNum() <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	txType := tx.Payload.TxType
	principal, err := p.CreatePrincipal(
		txType.String(),
		[]*commonPb.EndorsementEntry{tx.Sender},
		txBytes,
	)
	if err != nil {
		return false, fmt.Errorf("fail to construct authentication principal for %s : %s", txType.String(), err)
	}

	refinedPrincipal := principal
	if !bypassSignVerify {
		refinedPrincipal, err = p.refinePrincipal(principal)
		if err != nil {
			return false, fmt.Errorf("authentication failed, [%s]", err.Error())
		}
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	pol, err := p.lookUpPolicyByTxType(principal.GetResourceName(), blockVersion)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return p.verifyPrincipalPolicy(principal, refinedPrincipal, pol)
}

func verifySenderPrincipal(p acProvider, tx *commonPb.Transaction, txBytes []byte,
	blockVersion uint32, bypassVerifySign bool) (allow bool, err error) {

	if p.getTotalVoterNum() <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	resourceName := utils.GetTxResourceName(tx)
	principal, err := p.CreatePrincipal(
		resourceName,
		[]*commonPb.EndorsementEntry{tx.Sender},
		txBytes,
	)
	if err != nil {
		return false, fmt.Errorf("fail to construct authentication principal for %s : %s", resourceName, err)
	}

	refinedPrincipal := principal
	if !bypassVerifySign {
		refinedPrincipal, err = p.refinePrincipal(principal)
		if err != nil {
			return false, fmt.Errorf("authentication failed, [%s]", err.Error())
		}
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	pol, err := p.findFromSenderPolicies(principal.GetResourceName(), blockVersion)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}
	if pol == nil {
		return true, nil
	}

	return p.verifyPrincipalPolicy(principal, refinedPrincipal, pol)
}

func verifyEndorsementsPrincipal(p acProvider, tx *commonPb.Transaction, txBytes []byte,
	blockVersion uint32, bypassVerifySign bool) (allow bool, err error) {

	if p.getTotalVoterNum() <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	// 查找 resourceName 的策略
	resourceName := utils.GetTxResourceName(tx)
	pol, err := p.findFromEndorsementsPolicies(resourceName, blockVersion)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}
	if pol == nil {
		return true, nil
	}

	// 构建 endorsements
	endorsements := tx.Endorsers
	if endorsements == nil {
		endorsements = []*commonPb.EndorsementEntry{tx.Sender}
	}

	var principal protocol.Principal
	if pol.rule == protocol.RuleSelf {
		var targetOrg string
		parameterPairs := tx.Payload.Parameters
		if parameterPairs != nil {
			for i := 0; i < len(parameterPairs); i++ {
				key := parameterPairs[i].Key
				if key == protocol.ConfigNameOrgId {
					targetOrg = string(parameterPairs[i].Value)
					break
				}
			}
		}
		if targetOrg == "" {
			return false, fmt.Errorf("verification rule is [SELF], but org_id is not set in the parameter")
		}
		principal, err = p.CreatePrincipalForTargetOrg(resourceName, endorsements, txBytes, targetOrg)
	} else {
		principal, err = p.CreatePrincipal(resourceName, endorsements, txBytes)
	}
	if err != nil {
		return false, fmt.Errorf("fail to construct authentication principal for %s: %s",
			resourceName, err)
	}

	refinedPrincipal := principal
	if !bypassVerifySign {
		refinedPrincipal, err = p.refinePrincipal(principal)
		if err != nil {
			return false, fmt.Errorf("authentication failed, [%s]", err.Error())
		}
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	return p.verifyPrincipalPolicy(principal, refinedPrincipal, pol)
}

func verifyTxPrincipal(tx *commonPb.Transaction, resourceId string,
	p acProvider, blockVersion uint32) (bool, error) {
	var err error
	var allow bool
	var crossCall bool

	txBytes, err := utils.CalcUnsignedTxBytes(tx)
	if err != nil {
		return false, fmt.Errorf("get tx bytes failed, err = %v", err)
	}
	txType := tx.Payload.TxType
	txResourceId := utils.GetTxResourceName(tx)
	crossCall = false
	if txResourceId != resourceId {
		crossCall = true
	}

	// check tx_type
	allow, err = verifyTxTypePrincipal(p, tx, txBytes, blockVersion, crossCall)
	if err != nil {
		return false, fmt.Errorf("authentication error: %s", err)
	}
	if !allow {
		return false, fmt.Errorf("authentication failed")
	}

	if txType != commonPb.TxType_INVOKE_CONTRACT {
		return true, nil
	}

	// check sender

	allow, err = verifySenderPrincipal(p, tx, txBytes, blockVersion, crossCall)
	if err != nil {
		return false, fmt.Errorf("authentication error: %s", err)
	}
	if !allow {
		return false, fmt.Errorf("authentication failed")
	}

	// check endorsements
	allow, err = verifyEndorsementsPrincipal(p, tx, txBytes, blockVersion, crossCall)
	if err != nil {
		return false, fmt.Errorf("authentication error for %s: %s", resourceId, err)
	}
	if !allow {
		return false, fmt.Errorf("authentication failed for %s", resourceId)
	}

	return true, nil
}

func isRuleSupportedByMultiSign(
	p acProvider, resourceName string, blockVersion uint32, log protocol.Logger) error {
	policy, err := p.findFromEndorsementsPolicies(resourceName, blockVersion)
	if err != nil {
		// not found then there is no authority which means no need to sign multi sign
		log.Warn(err)
		return errors.New("this resource[" + resourceName + "] doesn't support to online multi sign")
	}
	if policy.GetRule() == protocol.RuleSelf {
		return errors.New("this resource[" + resourceName + "] is the self rule and doesn't support to online multi sign")
	}
	return nil
}

func isMultiSignRefused(p acProvider, resourceName string, rejects []*commonPb.EndorsementEntry,
	payload *commonPb.Payload, blockVersion uint32, log protocol.Logger) (bool, error) {

	totalVotes := p.getTotalVoterNum()
	data, err := payload.Marshal()
	if err != nil {
		return false, fmt.Errorf("marshal MultiSignInfo.Payload failed, err = %v", err)
	}

	refinedRejects := p.RefineEndorsements(rejects, data)
	policy, err := p.findFromEndorsementsPolicies(resourceName, blockVersion)
	if err != nil {
		return false, err
	}

	switch policy.GetRule() {
	case protocol.RuleForbidden:
		log.Infof("policy of multi-sign tx should not be `%v`", protocol.RuleForbidden)
		return false, fmt.Errorf("policy of multi-sign tx should not be `%v`", protocol.RuleForbidden)

	case protocol.RuleAny:
		if len(refinedRejects) == totalVotes {
			log.Infof("rule = %v, %d rejects make multi-sign tx failed.", protocol.RuleAny, len(refinedRejects))
			return true, nil
		}
		log.Infof("rule = %v, multi-sign tx has %d/%d validate rejects", protocol.RuleAny, len(refinedRejects), totalVotes)
		return false, nil

	case protocol.RuleMajority:
		if 2*len(refinedRejects) >= totalVotes {
			log.Infof("rule = %v, %d rejects make multi-sign tx failed.", protocol.RuleMajority, len(refinedRejects))
			return true, nil
		}
		log.Infof("rule = %v, multi-sign tx has less than half validate rejects", protocol.RuleMajority)
		return false, nil

	case protocol.RuleAll:
		if len(refinedRejects) > 0 {
			log.Infof("rule = %v, %d rejects make multi-sign tx failed.", protocol.RuleAll, len(refinedRejects))
			return true, nil
		}
		log.Infof("rule = %v, multi-sign tx has no validate rejects", protocol.RuleAll)
		return false, nil

	case protocol.RuleSelf:
		return false, fmt.Errorf("unsupported policy `%v`", protocol.RuleSelf)

	default:
		return false, fmt.Errorf("unsupported policy `%v`", policy.GetRule())
	}
}

func verifyMultiSignTxPrincipal(p acProvider, mInfo *syscontract.MultiSignInfo,
	blockVersion uint32, log protocol.Logger) (syscontract.MultiSignStatus, error) {

	if mInfo.Status != syscontract.MultiSignStatus_PROCESSING {
		return mInfo.Status, fmt.Errorf("multi-sign status `%v` is not permitted to verify", mInfo.Status)
	}

	resourceName := mInfo.Payload.ContractName + "-" + mInfo.Payload.Method

	agreeEndorsements := make([]*commonPb.EndorsementEntry, len(mInfo.VoteInfos))
	rejectEndorsements := make([]*commonPb.EndorsementEntry, len(mInfo.VoteInfos))

	for _, voteInfo := range mInfo.VoteInfos {
		if voteInfo.Vote == syscontract.VoteStatus_AGREE {
			agreeEndorsements = append(agreeEndorsements, voteInfo.Endorsement)
		} else if voteInfo.Vote == syscontract.VoteStatus_REJECT {
			rejectEndorsements = append(rejectEndorsements, voteInfo.Endorsement)
		} else {
			log.Warnf("unknown vote action, voteInfo.Vote = %v", voteInfo.Vote)
		}
	}
	log.Debugf("endorsers agreed num => %v", len(agreeEndorsements))
	log.Debugf("endorsers rejected num => %v", len(rejectEndorsements))

	// 根据 agree 的数量判断多签状态
	if len(agreeEndorsements) > 0 {
		tx := &commonPb.Transaction{
			Payload:   mInfo.Payload,
			Endorsers: agreeEndorsements,
		}
		agree, err := verifyTxPrincipal(tx, resourceName, p, blockVersion)
		if err != nil {
			return mInfo.Status, err
		}
		if agree {
			return syscontract.MultiSignStatus_PASSED, nil
		}
	}

	// 根据 reject 的数量判断多签状态
	refuse, err := isMultiSignRefused(p, resourceName, rejectEndorsements, mInfo.Payload, blockVersion, log)
	if err != nil {
		return mInfo.Status, err
	}

	if refuse {
		return syscontract.MultiSignStatus_REFUSED, nil
	}

	return mInfo.Status, nil
}
