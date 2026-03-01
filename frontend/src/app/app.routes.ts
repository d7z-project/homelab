import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login.component';
import { MainComponent } from './pages/main/main.component';
import { WelcomeComponent } from './pages/welcome/welcome.component';
import { RbacComponent } from './pages/rbac/rbac.component';
import { RbacSimulatorComponent } from './pages/rbac/simulator.component';
import { AuditComponent } from './pages/audit/audit.component';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  {
    path: '',
    component: MainComponent,
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'welcome' },
      { path: 'welcome', component: WelcomeComponent },
      { path: 'rbac', component: RbacComponent },
      { path: 'rbac/simulator', component: RbacSimulatorComponent },
      { path: 'audit', component: AuditComponent },
    ],
  },
];
