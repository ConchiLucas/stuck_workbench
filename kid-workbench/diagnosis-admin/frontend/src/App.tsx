import { Navigate, Route, Routes } from 'react-router-dom'
import { NavigationPosition } from './lib/listNavigation'
import { AppShell } from './layout/AppShell'
import { KnowledgeOverviewPage } from './pages/KnowledgeOverviewPage'
import { LibraryPage } from './pages/LibraryPage'
import { WrongAnswersPage } from './pages/WrongAnswersPage'
import { AttemptDetailPage } from './pages/AttemptDetailPage'
import { ReviewSuggestionsPage } from './pages/ReviewSuggestionsPage'
import { ReviewSuggestionPage } from './pages/ReviewSuggestionPage'

import { KpArchivePage } from './pages/KpArchivePage'


export function App() {
  return (
    <AppShell>
      <NavigationPosition/>
      <Routes>
        <Route path="/" element={<KnowledgeOverviewPage />} />
        <Route path="/library" element={<LibraryPage />} />
        <Route path="/subjects/game" element={<Navigate to="/" replace />} />
        <Route path="/subjects/:code" element={<LibraryPage />} />
        <Route path="/knowledge-points/:kpId" element={<KpArchivePage />} />
        <Route path="/patterns" element={<Navigate to="/wrongs?view=groups" replace />} />
        <Route path="/wrongs" element={<WrongAnswersPage />} />
        <Route path="/attempts/:attemptId" element={<AttemptDetailPage />} />
        <Route path="/reviews" element={<ReviewSuggestionsPage />} />
        <Route path="/reviews/:id" element={<ReviewSuggestionPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppShell>
  )
}
