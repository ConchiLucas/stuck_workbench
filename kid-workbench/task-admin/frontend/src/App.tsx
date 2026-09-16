import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './layout/AppShell'
import { EnglishTasksPage } from './pages/EnglishTasksPage'
import { GenerationPage } from './pages/GenerationPage'
import { MathTasksPage } from './pages/MathTasksPage'
import { PinyinTasksPage } from './pages/PinyinTasksPage'
import { PhraseTasksPage } from './pages/PhraseTasksPage'
import { ChengyuTasksPage } from './pages/ChengyuTasksPage'
import { LogicTasksPage } from './pages/LogicTasksPage'
import { PoemTasksPage } from './pages/PoemTasksPage'
import { ScienceTasksPage } from './pages/ScienceTasksPage'
import { TasksPage } from './pages/TasksPage'

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<GenerationPage />} />
        <Route path="/tasks/:id" element={<GenerationPage />} />
        <Route path="/pinyin" element={<PinyinTasksPage />} />
        <Route path="/math" element={<MathTasksPage />} />
        <Route path="/english" element={<EnglishTasksPage />} />
        <Route path="/phrase" element={<PhraseTasksPage />} />
        <Route path="/chengyu" element={<ChengyuTasksPage />} />
        <Route path="/science" element={<ScienceTasksPage />} />
        <Route path="/poem" element={<PoemTasksPage />} />
        <Route path="/logic" element={<LogicTasksPage />} />
        <Route path="/legacy" element={<TasksPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppShell>
  )
}
