export * from './auth.service';
import { AuthService } from './auth.service';
export * from './example.service';
import { ExampleService } from './example.service';
export * from './rbac.service';
import { RbacService } from './rbac.service';
export const APIS = [AuthService, ExampleService, RbacService];
