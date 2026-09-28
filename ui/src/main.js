import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from '~/App.vue'
import Overview from '~/pages/Overview.vue'
import Graph from '~/pages/Graph.vue'
import Knowledge from '~/pages/Knowledge.vue'
import Repositories from '~/pages/Repositories.vue'
import Repository from '~/pages/Repository.vue'
import Reviews from '~/pages/Reviews.vue'
import Skill from '~/pages/Skill.vue'
import SkillForm from '~/pages/SkillForm.vue'
import Skills from '~/pages/Skills.vue'
import SettingsPage from '~/pages/Settings.vue'
import '@sourceant/design/tokens.css'

// Real paths, so a link somebody is handed can be opened and pasted like any
// other URL. The agent answers anything that is not a file with the page.
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Overview },
    { path: '/graph', component: Graph },
    { path: '/knowledge', component: Knowledge },
    { path: '/reviews', component: Reviews },
    // A review has a name, so an agent can hand somebody a link to one.
    { path: '/reviews/:id', component: Reviews },
    { path: '/skills', component: Skills },
    // Writing one and changing one are the same form, entered on purpose.
    { path: '/skills/new', component: SkillForm },
    { path: '/skills/:id(.*)/edit', component: SkillForm },
    // A rule kept in a nested folder has a slash in its name.
    { path: '/skills/:id(.*)', component: Skill },
    { path: '/repositories', component: Repositories },
    // A name has a slash in it, so the whole tail is the name.
    { path: '/repositories/:name(.*)', component: Repository },
    { path: '/settings', component: SettingsPage },
    { path: '/:rest(.*)*', redirect: '/' },
  ],
})

createApp(App).use(router).mount('#app')
