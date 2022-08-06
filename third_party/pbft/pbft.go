package pbft

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
)

//本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool = []Message{}

type NodeInfo struct {
	//节点ID
	NodeID string
	//RSA私钥
	RsaPrivKey []byte
	//RSA公钥
	RsaPubKey   []byte
	NodePubKeys [][]byte
	NodeNum     int
}

type Pbft struct {
	//节点信息
	node NodeInfo
	//每笔请求自增序号
	sequenceID int
	//锁
	lock sync.Mutex
	//临时消息池，消息摘要对应消息本体
	messagePool map[string]Request
	//存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	prePareConfirmCount map[string]map[string]bool
	//存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	commitConfirmCount map[string]map[string]bool
	//该笔消息是否已进行Commit广播
	isCommitBordcast map[string]bool
	//该笔消息是否已对客户端进行Reply
	isReply map[string]bool
}

func NewPBFT(nodeInfo NodeInfo) *Pbft {
	p := new(Pbft)
	p.node = nodeInfo
	p.sequenceID = 0
	p.messagePool = make(map[string]Request)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	return p
}

//处理客户端发来的请求
func (p *Pbft) HandleClientRequest(r *Request, broadcast func(string, any)) {
	fmt.Println("主节点已接收到客户端发来的request ...")
	//添加信息序号
	p.sequenceIDAdd()
	//获取消息摘要
	digest := getDigest(*r)
	fmt.Println("已将request存入临时消息池")
	//存入临时消息池
	p.messagePool[digest] = *r
	//主节点对消息摘要进行签名
	digestByte, _ := hex.DecodeString(digest)
	signInfo := p.RsaSignWithSha256(digestByte, p.node.RsaPrivKey)
	//拼接成PrePrepare，准备发往follower节点
	pp := PrePrepare{*r, digest, p.sequenceID, signInfo}
	//进行PrePrepare广播
	fmt.Println("正在向其他节点进行进行PrePrepare广播 ...")
	broadcast("preprepare", pp)
	fmt.Println("PrePrepare广播完成")
}

//处理预准备消息
func (p *Pbft) HandlePrePrepare(pp *PrePrepare, broadcast func(string, any)) {
	fmt.Println("本节点已接收到主节点发来的PrePrepare ...")
	//获取主节点的公钥，用于数字签名验证
	primaryNodePubKey := p.getPubKey("0")
	digestByte, _ := hex.DecodeString(pp.Digest)
	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("信息摘要对不上，拒绝进行prepare广播")
	} else if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("消息序号对不上，拒绝进行prepare广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		fmt.Println("主节点签名验证失败！,拒绝进行prepare广播")
	} else {
		//序号赋值
		p.sequenceID = pp.SequenceID
		//将信息存入临时消息池
		fmt.Println("已将消息存入临时节点池")
		p.messagePool[pp.Digest] = pp.RequestMessage
		//节点使用私钥对其签名
		sign := p.RsaSignWithSha256(digestByte, p.node.RsaPrivKey)
		//拼接成Prepare
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.NodeID, sign}
		//进行准备阶段的广播
		fmt.Println("正在进行Prepare广播 ...")
		broadcast("prepare", pre)
		fmt.Println("Prepare广播完成")
	}
}

//处理准备消息
func (p *Pbft) HandlePrepare(pre *Prepare, broadcast func(string, any)) {
	fmt.Printf("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if p.sequenceID != pre.SequenceID {
		fmt.Println("消息序号对不上，拒绝执行commit广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
	} else {
		p.lock.Lock()
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//因为主节点不会发送Prepare，所以不包含自己
		specifiedCount := 0
		if p.node.NodeID == "0" {
			specifiedCount = p.node.NodeNum / 3 * 2
		} else {
			specifiedCount = (p.node.NodeNum / 3 * 2) - 1
		}
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		//获取消息源节点的公钥，用于数字签名验证
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			fmt.Println("本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")
			//节点使用私钥对其签名
			sign := p.RsaSignWithSha256(digestByte, p.node.RsaPrivKey)
			c := Commit{pre.Digest, pre.SequenceID, p.node.NodeID, sign}
			//进行提交信息的广播
			fmt.Println("正在进行commit广播")
			broadcast("commit", c)
			p.isCommitBordcast[pre.Digest] = true
			fmt.Println("commit广播完成")
		}
		p.lock.Unlock()
	}
}

//处理提交确认消息
func (p *Pbft) HandleCommit(c *Commit, reply func(Message)) {
	fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		fmt.Println("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
	} else if p.sequenceID != c.SequenceID {
		fmt.Println("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else if !p.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else {
		p.lock.Lock()
		p.setCommitConfirmMap(c.Digest, c.NodeID, true)
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		if count >= p.node.NodeNum/3*2 && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
			msg := p.messagePool[c.Digest].Message
			fmt.Println("正在reply客户端 ...")
			reply(msg)
			p.isReply[c.Digest] = true
			fmt.Println("reply完毕")
		}
		p.lock.Unlock()
	}
}

//序号累加
func (p *Pbft) sequenceIDAdd() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.sequenceID++
}

//为多重映射开辟赋值
func (p *Pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

//为多重映射开辟赋值
func (p *Pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok { //TODO
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

//传入节点编号， 获取对应的公钥
func (p *Pbft) getPubKey(nodeID string) []byte {
	i, err := strconv.Atoi(nodeID)
	if err != nil {
		panic(err)
	}

	return p.node.NodePubKeys[i]
}
