import { useLayoutEffect } from 'react'
import { Link, useLocation, useNavigationType } from 'react-router-dom'
import { getChildId } from '../store/childStore'

type ReturnState={listReturn?:{to:string};restoreList?:boolean}
const positions=new Map<string,number>()
export function useListReturn() {
 const {pathname,search}=useLocation()
 return ()=>({listReturn:{to:pathname+search}})
}
export function BackToList({fallback,label}:{fallback:string;label:string}) {
 const {state}=useLocation(); const target=(state as ReturnState|null)?.listReturn
 const safe=target&&/^\/(library|subjects|wrongs|knowledge-points|reviews)([/?]|$)/.test(target.to)
 return <Link to={safe?target.to:fallback} state={safe?{restoreList:true}:undefined}>{label}</Link>
}
export function NavigationPosition() {
 const location=useLocation(), type=useNavigationType(),child=getChildId()
 useLayoutEffect(()=>{
  const key=`${child}:${location.pathname}${location.search}`
  const desired=((location.state as ReturnState|null)?.restoreList||type==='POP'?positions.get(key):0)??0
  let frame=0,tries=0,restoring=true
  const track=()=>{if(!restoring)positions.set(key,window.scrollY)}
  const restore=()=>{window.scrollTo({top:desired,behavior:'instant'});if(document.documentElement.scrollHeight-window.innerHeight>=desired||tries++>60){restoring=false}else{frame=requestAnimationFrame(restore)}}
  frame=requestAnimationFrame(restore)
  window.addEventListener('scroll',track,{passive:true})
  return ()=>{cancelAnimationFrame(frame);window.removeEventListener('scroll',track);if(!restoring)positions.set(key,window.scrollY)}
 },[child,location.key,location.pathname,location.search,type])
 return null
}
