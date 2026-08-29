import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from '~/App.vue'
import Overview from '~/pages/Overview.vue'
import Graph from '~/pages/Graph.vue'
import Knowledge from '~/pages/Knowledge.vue'
import Repositories from '~/pages/Repositories.vue'
import Repository from '~/pages/Repository.vue'
import Reviews from '~/pages/Reviews.vue'
import Skill from '~/pages/Skill.vue'
import Skills from '~/pages/Skills.vue'
import SettingsPage from '~/pages/Settings.vue'
import '@sourceant/design/tokens.css'

// Hash history, because the agent serves one file and knows nothing about
// paths a router invented.
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Overview },
    { path: '/graph', component: Graph },
    { path: '/knowledge', component: Knowledge },
    { path: '/reviews', component: Reviews },
    { path: '/skills', component: Skills },
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
