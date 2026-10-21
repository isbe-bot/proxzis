import { createRouter } from 'vue-router'
import HomeView from './views/HomeView.vue'
import MatchesView from './views/MatchesView.vue'
import MatchView from './views/MatchView.vue'
import AgentView from './views/AgentView.vue'
import HouseView from './views/HouseView.vue'
import AboutView from './views/AboutView.vue'
import RegisterView from './views/RegisterView.vue'
import NotFoundView from './views/NotFoundView.vue'

export const routes = [
  { path: '/', name: 'home', component: HomeView, meta: { title: 'The arena for autonomous agents' } },
  { path: '/matches', name: 'matches', component: MatchesView, meta: { title: 'Matches' } },
  { path: '/matches/:id', name: 'match', component: MatchView, props: true, meta: { title: 'Match replay' } },
  { path: '/agents/:id', name: 'agent', component: AgentView, props: true, meta: { title: 'Agent' } },
  { path: '/house', name: 'house', component: HouseView, meta: { title: 'The house' } },
  { path: '/about', name: 'about', component: AboutView, meta: { title: 'About' } },
  { path: '/register', name: 'register', component: RegisterView, meta: { title: 'List your agent' } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView, meta: { title: 'Not found' } },
]

export function createAppRouter(history) {
  return createRouter({
    history,
    routes,
    scrollBehavior(to, from, saved) {
      if (saved) return saved
      if (to.hash) {
        const reduce = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
        return { el: to.hash, behavior: reduce ? 'auto' : 'smooth' }
      }
      return { top: 0 }
    },
  })
}
