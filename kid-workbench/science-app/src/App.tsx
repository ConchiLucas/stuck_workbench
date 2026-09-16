import { appBase } from './appPath'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { HomePage } from './pages/HomePage'
import { LearnPage } from './pages/LearnPage'
import { MapPage } from './pages/MapPage'
import { PracticePage } from './pages/PracticePage'
import { QuestionTypePage } from './pages/QuestionTypePage'
import { QuestionTypeResultPage } from './pages/QuestionTypeResultPage'
import { ResultPage } from './pages/ResultPage'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } })

export function AppRoutes() {
  return <QueryClientProvider client={queryClient}><Routes><Route element={<Shell />}>
    <Route index element={<HomePage />} />
    <Route path="explore" element={<MapPage />} />
    <Route path="concept/:kpId" element={<LearnPage />} />
    <Route path="question-types/:slug/result" element={<QuestionTypeResultPage />} />
    <Route path="question-types/:slug/:questionId" element={<QuestionTypePage />} />
    <Route path="question-types/:slug" element={<QuestionTypePage />} />
    <Route path="practice/:planId" element={<PracticePage />} />
    <Route path="practice/:planId/done" element={<ResultPage />} />
  </Route></Routes></QueryClientProvider>
}

export default function App() { return <BrowserRouter basename={appBase() || "/"}><AppRoutes /></BrowserRouter> }
