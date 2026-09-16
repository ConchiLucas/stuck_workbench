import { appBase } from './appPath'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { TasksPage } from './pages/TasksPage'
import { HomePage } from './pages/HomePage'
import { DemoPracticePage } from './pages/DemoPracticePage'
import { CharPage } from './pages/CharPage'
import { LearnPage } from './pages/LearnPage'
import { MapPage } from './pages/MapPage'
import { PracticePage } from './pages/PracticePage'
import { ResultPage } from './pages/ResultPage'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } })

export function AppRoutes() {
  return <QueryClientProvider client={queryClient}><Routes><Route element={<Shell />}>
    <Route index element={<HomePage />} />
    <Route path="tasks" element={<TasksPage />} />
    <Route path="map" element={<MapPage />} />
    <Route path="char/:character" element={<CharPage />} />
    <Route path="learn/:kpId" element={<LearnPage />} />
    <Route path="demo/:type" element={<DemoPracticePage />} />
    <Route path="practice/type/:type" element={<DemoPracticePage />} />
    <Route path="practice/:planId" element={<PracticePage />} />
    <Route path="practice/:planId/result" element={<ResultPage />} />
  </Route></Routes></QueryClientProvider>
}

export default function App() { return <BrowserRouter basename={appBase() || "/"}><AppRoutes /></BrowserRouter> }
