import type { EnglishExample } from '@kid-workbench/english-player'
import type { MathExample } from '@kid-workbench/math-player'
import type { PhraseExample } from '@kid-workbench/phrase-player'
import type { ChengyuExample } from '@kid-workbench/chengyu-player'
import type { ScienceExample } from '@kid-workbench/science-player'
import type { PoemExample } from '@kid-workbench/poem-player'
import type { LogicExample } from '@kid-workbench/logic-player'
export type Stats={observedAttempts:number;observedCorrect:number;independentAttempts:number;independentCorrect:number;assistedAttempts:number;unknownAssistanceAttempts:number;accuracy:number|null;independentAccuracy:number|null}
export type Skill={skillCode:string;label:string;masteryStatus:string;practiced:boolean;stats:Stats;evidenceCoverage:string}
export type Coverage={level:string;reasonCodes:string[]}
export type Point={masteryCategory?:string;reviewDue?:boolean;firstMasteredAt?:string|null;kpId:number;title:string;subjectCode:string;subjectName:string;moduleCode:string;moduleName:string;masteryStatus:string;stats:Stats;skills:Skill[];wrongCount:number;hasMasteredAbility:boolean;coverage:Coverage}
export type AbilityCounts={mastered:number;learning:number;shaky:number;unpracticed:number;unknown:number}
export type ReviewStatusSummary={pending:number;draft:number;awaiting:number;answered:number;total:number}
export type Subject={pointCounts?:PointCounts;attemptsCount?:number;code:string;name:string;total:number;practicedCount:number;masteredAbilityCount:number;wrongPointCount:number;wrongCount:number;abilities:AbilityCounts;unmappedPointCount?:number;coverage:Coverage}
export type Summary={totalCount?:number;pointCounts?:PointCounts;child:{id:number;name:string;grade:string};subjects:Subject[];stats:Stats;practicedCount:number;masteredAbilityCount:number;wrongPointCount:number;wrongCount:number}
export type Page<T>={items:T[];nextCursor?:string;hasMore:boolean;evidenceAsOf:string;stateReadAt:string;coverage?:Coverage}
export type SourceRef={kind:string;receiptId?:number;questionVersionId?:number;instanceId?:string;clientId?:string;planId?:number;itemId?:number}
export type Option={id:string;label:string;semanticId?:string;imageMediaId?:string;audioMediaId?:string}
export type Question={visual?:{kind?:string;text?:string;initial?:string;final?:string;letter?:string;syllable?:string};interaction:string;targetText:string;stem:{text:string;imageMediaId?:string;audioMediaId?:string};options:Option[];answerOptionId?:string}
export type Evidence={attemptId:number;childId:number;kpId:number;title:string;subjectCode:string;subjectName:string;moduleName:string;moduleCode:string;skillCode:string;skillLabel:string;questionType:string;occurredAt:string;isCorrect:boolean;costMs:number;assistance:string;source:SourceRef;questionFidelity:string;selectionFidelity:string;mediaFidelity:string;question:Question|null;mathExample?:MathExample;englishExample?:EnglishExample;phraseExample?:PhraseExample;chengyuExample?:ChengyuExample;scienceExample?:ScienceExample;poemExample?:PoemExample;logicExample?:LogicExample;audioMutable?:boolean;response:{kind:string;selectedOptionId?:string;value?:string;strokes?:unknown;hintsUsed?:number;evaluation?:{outcome?:string};evaluatorVersion?:string};evidenceReasonCodes:string[];reviewEligible:boolean;reviewBlockReasons:string[];followUpState:string}
export type Candidate={key:string;kpId:number;title:string;subjectCode:string;moduleCode:string;skillCode:string;questionType:string;reasonCode:string;reasonText:string;wrongCount:number;independentInstanceCount:number;wrongInitialCount:number;lastWrongAt:string;evidence:{attemptId:number;role:string;source:SourceRef}[];mode:string;requestedCount:number;preferredDistractorKpIds:number[];reviewEligible:boolean;reviewBlockReasons:string[]}
export type Target={title?:string;id?:number;key:string;kpId:number;questionType:string;reasonCode:string;reasonText:string;mode:string;requestedCount:number}
export type Partition={id:number;key:string;status:string;taskId?:number;revisionId?:number;generatedCount:number;error?:{code:string;message:string;retryable:boolean}}
export type Suggestion={id:number;title:string;childId:number;subjectCode:string;rowVersion:number;lifecycle:string;generationStatus:string|null;requestedCount:number;generatedCount:number;taskCount:number;targets:Target[];partitions:Partition[];createdAt:string}
export type TaskLink={taskId:number;revisionId?:number;generatedRevisionId?:number;status?:string;taskStatus?:string;title?:string}
export type Outcome={revisionChangedPlans:number;key:string;attemptIds:number[];evidenceTruncated:boolean;unknownAssistanceAttempts:number;kpId:number;questionType:string;title:string;state:string;observedAttempts:number;independentCorrect:number;wrongCount:number;assistedAttempts:number;originalAttempts:number;variantAttempts:number;masteryStatus:string;otherLaterAttempts:number}

export type PointCounts={complete:number;partial:number;weak:number;learning:number;unpracticed:number;unknown:number;reviewDue:number}
export type CalendarSubject={code:string;name:string;mastered:number;total:number;wrong:number}
export type CalendarDay={date:string;mastered:number;total:number;wrong:number;subjects:CalendarSubject[]}
export type KnowledgeCalendar={timezone:string;today:string;month:string;days:CalendarDay[];trend:CalendarDay[];masteryDateUnknownCount:number}
