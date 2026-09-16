import {create} from 'zustand'

type DemoAnswerStore={
  picks:Record<string,string>
  setPick:(type:string,n:number,value:string)=>void
  pick:(type:string,n:number)=>string|undefined
  clearType:(type:string)=>void
}

function key(type:string,n:number){
  return `${type}:${n}`
}

export const useDemoAnswerStore=create<DemoAnswerStore>((set,get)=>({
  picks:{},
  setPick:(type,n,value)=>set(state=>({picks:{...state.picks,[key(type,n)]:value}})),
  pick:(type,n)=>get().picks[key(type,n)],
  clearType:(type)=>set(state=>{
    const picks={...state.picks}
    for(const name of Object.keys(picks)){
      if(name.startsWith(`${type}:`)) delete picks[name]
    }
    return {picks}
  }),
}))
