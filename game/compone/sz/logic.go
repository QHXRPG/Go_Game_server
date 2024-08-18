package sz

import (
	"common/utils"
	"sort"
	"sync"
)

type Logic struct {
	sync.RWMutex
	cards []int //52张牌
}

func NewLogic() *Logic {
	return &Logic{
		cards: make([]int, 0),
	}
}

// washCards
func (l *Logic) washCards() {
	// 4个花色，每个花色13张牌
	l.cards = []int{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, // 方块
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, // 梅花
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, // 红桃
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, // 黑桃
	}
	for i, v := range l.cards {
		random := utils.Rand(len(l.cards))
		l.cards[i] = l.cards[random]
		l.cards[random] = v
	}
}

// getCards 获取三张手牌
func (l *Logic) getCards() []int {
	// 3 * userNums 的二维切片，存储每位玩家所发到的手牌
	cards := make([]int, 3)
	l.RLock()
	defer l.RUnlock()
	for i := 0; i < 3; i++ {
		if len(cards) == 0 {
			break
		}
		card := l.cards[len(l.cards)-1]
		l.cards = l.cards[:len(l.cards)-1]
		cards[i] = card
	}
	return cards
}

// CompareCards  0:和局、>0:胜利 、<0:失败
func (l *Logic) CompareCards(from []int, to []int) int {
	//获取牌的类型
	fromType := l.getCardsType(from)
	toType := l.getCardsType(to)
	// 如果手牌类型不相等，比较谁的手牌类型更大
	if fromType != toType {
		return int(fromType - toType)
	}
	// 类型相等，需要进行比较牌面大小
	// 如果牌类型是对子，需要判断对子大小
	if fromType == DuiZi {
		duiFrom, danFrom := l.getDuiZi(from)
		duiTo, danTo := l.getDuiZi(to)
		if duiFrom != duiTo {
			return duiFrom - duiTo
		} else {
			return danFrom - danTo
		}
	}
	// 比较牌面大小
	valuesFrom := l.getCardValues(from)
	valuesTo := l.getCardValues(to)
	if valuesFrom[2] != valuesTo[2] {
		return valuesFrom[2] - valuesTo[2]
	}
	if valuesFrom[1] != valuesTo[1] {
		return valuesFrom[1] - valuesTo[1]
	}
	if valuesFrom[0] != valuesTo[0] {
		return valuesFrom[0] - valuesTo[0]
	}
	return 0
}

// 解析手牌的类型
func (l *Logic) getCardsType(cards []int) CardsType {
	// 1. 豹子：牌面值相等
	one := l.getCardsValue(cards[0])
	two := l.getCardsValue(cards[1])
	three := l.getCardsValue(cards[2])
	if one == two && two == three {
		return BaoZi
	}
	// 2. 金花：颜色相同+顺子
	jinhua := false
	oneColor := l.getCardsColor(cards[0])
	twoColor := l.getCardsColor(cards[1])
	threeColor := l.getCardsColor(cards[2])
	if oneColor == twoColor && twoColor == threeColor {
		jinhua = true
	}
	// 3. 顺子：排序后看满不满足顺子条件
	shunzi := false
	values := l.getCardValues(cards) // 将手牌排序
	oneValue := values[0]
	twoValue := values[1]
	threeValue := values[2]
	if oneValue+1 == twoValue && twoValue+1 == threeValue {
		shunzi = true
	}
	if oneValue == 2 && twoValue == 3 && threeValue == 14 {
		shunzi = true
	}
	if jinhua && shunzi {
		return ShunJin
	}
	if jinhua {
		return JinHua
	}
	if shunzi {
		return ShunJin
	}
	if oneValue == twoValue || twoValue == threeValue {
		return DuiZi
	}
	return DanZhang
}

// 将传过来的手牌排序
func (l *Logic) getCardValues(cards []int) []int {
	v := make([]int, len(cards))
	for i, card := range cards {
		v[i] = l.getCardsValue(card)
	}
	sort.Ints(v)
	return v
}

func (l *Logic) getCardsValue(card int) int {
	value := card & 0x0f
	// 如果当前的牌是A，它的值应该是14
	if value == 1 {
		value += 13
	}
	return value
}

func (l *Logic) getCardsColor(card int) string {
	color := []string{"方块", "梅花", "红桃", "黑桃"}
	// 取模得到牌色
	if card >= 0x01 && card <= 0x03 {
		return color[card/0x10] // 因为是16进制，所以除以0x10(16)
	}
	return ""
}

// 拿到对子的大小
func (l *Logic) getDuiZi(cards []int) (int, int) {
	values := l.getCardValues(cards)
	if values[0] == values[1] {
		return values[0], values[2]
	}
	return values[1], values[0]
}
