export type InteractionFamily='choice'|'order'|'input'

export interface QuestionType {
  id:string
  code:string
  category:string
  title:string
  kidTitle:string
  action:string
  goal:string
  family:InteractionFamily
  coveredTypes:string[]
  diagnosis:string[]
  nextStep:string
  ordinal:number
}

export const questionTypes:QuestionType[]=[
  {
    id:'audio-choice',code:'LISTEN',category:'输入理解',title:'音频选择脚本',kidTitle:'听音选词',
    action:'播放单词或句子，让孩子从图片或文字中选择答案',goal:'判断声音是否真正听懂',family:'choice',
    coveredTypes:['听音选图','听音选词','听句选图','听问句选回答'],
    diagnosis:['图片也选错：声音和词义还没有建立连接','图片正确、文字错误：听得懂，但不熟悉单词字形','句子题错误：需要先补关键词听力，再练整句理解'],
    nextStep:'先用熟悉图片建立声音—含义连接，再切换到英文选项，最后增加短句。',ordinal:1,
  },
  {
    id:'image-text',code:'LOOK',category:'词义连接',title:'图文双向选择脚本',kidTitle:'看图选词',
    action:'在图片与英文之间进行双向选择',goal:'区分“认得”与“主动想得起”',family:'choice',
    coveredTypes:['看图选词','看词选图','图文配句','读句选图'],
    diagnosis:['看词选图错：看到英文后无法提取词义','看图选词错：知道物品，但英文词形记忆不牢','单词题正确、句子题错误：需要进入基础句型练习'],
    nextStep:'先做英文到图片，再反向做图片到英文；双向都稳定后进入句子。',ordinal:2,
  },
  {
    id:'card-builder',code:'BUILD',category:'结构组合',title:'卡片组合脚本',kidTitle:'组句子',
    action:'点击或拖动音素、单词卡片完成组合',goal:'观察孩子如何组织语言结构',family:'order',
    coveredTypes:['CVC 拼读','拆分音素','乱序字母组词','单词排成句'],
    diagnosis:['音素顺序错误：合音和拆音关系不稳定','单词能拼、句子排错：句型结构尚未形成','反复试错：需要减少卡片数量并提供结构提示'],
    nextStep:'从三个音素的 CVC 组合开始，再升级到单词和四词短句。',ordinal:3,
  },
  {
    id:'input-gap',code:'TYPE',category:'独立输出',title:'输入填空脚本',kidTitle:'写单词',
    action:'让孩子补全或独立输入单词和句子',goal:'判断是否能从识别走向独立输出',family:'input',
    coveredTypes:['缺字母填空','听写单词','看图写单词','选词补句','听句填空'],
    diagnosis:['会选择但不会输入：词形仍停留在被动识别','固定位置漏字母：对应拼写模式没有掌握','单词正确、句子填错：需要补充语境和句型训练'],
    nextStep:'先给首尾字母提示，再逐步减少提示，最后过渡到完整听写。',ordinal:4,
  },
  {
    id:'reading-qa',code:'READ',category:'阅读理解',title:'阅读问答脚本',kidTitle:'读一读',
    action:'阅读短句或短文后选择、判断或排序',goal:'区分解码问题和理解问题',family:'choice',
    coveredTypes:['句子判断','短文选标题','阅读找答案','故事图片排序'],
    diagnosis:['读不出关键词：先回到拼读和高频词','能读出但答错：需要训练句意和信息定位','事实题正确、排序题错误：事件关系理解不足'],
    nextStep:'从一句话配图开始，过渡到两三句短文和故事顺序。',ordinal:5,
  },
]
