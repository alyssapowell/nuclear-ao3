// Every page of the site and the address it lives at. To add a page, create
// it in src/pages/ and add a line here. Pages inside <LoggedIn> send
// logged-out visitors to the login page first.
import { lazy, Suspense } from 'react'
import { Outlet, Route, Routes } from 'react-router-dom'
import Navigation from '@/components/Navigation'
import Footer from '@/components/Footer'
import GDPRConsentBanner from '@/components/GDPRConsentBanner'
import ServiceStatus from '@/components/ServiceStatus'
import TrialBanner from '@/components/TrialBanner'
import PWAInit from '@/components/PWAInit'
import AuthGuard from '@/components/auth/AuthGuard'

// Each page is downloaded only when someone first visits it, so the first
// visit to the site stays fast.
const Home = lazy(() => import('@/pages/Home'))
const About = lazy(() => import('@/pages/About'))
const Accessibility = lazy(() => import('@/pages/Accessibility'))
const ApiDocs = lazy(() => import('@/pages/ApiDocs'))
const Attributions = lazy(() => import('@/pages/Attributions'))
const Contact = lazy(() => import('@/pages/Contact'))
const Feedback = lazy(() => import('@/pages/Feedback'))
const Help = lazy(() => import('@/pages/Help'))
const Privacy = lazy(() => import('@/pages/Privacy'))
const Terms = lazy(() => import('@/pages/Terms'))
const Login = lazy(() => import('@/pages/Login'))
const Register = lazy(() => import('@/pages/Register'))
const Activate = lazy(() => import('@/pages/Activate'))
const ForgotPassword = lazy(() => import('@/pages/ForgotPassword'))
const ResetPassword = lazy(() => import('@/pages/ResetPassword'))
const ChangePassword = lazy(() => import('@/pages/ChangePassword'))
const Search = lazy(() => import('@/pages/Search'))
const WorkList = lazy(() => import('@/pages/WorkList'))
const WorkView = lazy(() => import('@/pages/WorkView'))
const WorkNew = lazy(() => import('@/pages/WorkNew'))
const WorkEdit = lazy(() => import('@/pages/WorkEdit'))
const SeriesList = lazy(() => import('@/pages/SeriesList'))
const SeriesView = lazy(() => import('@/pages/SeriesView'))
const SeriesNew = lazy(() => import('@/pages/SeriesNew'))
const SeriesEdit = lazy(() => import('@/pages/SeriesEdit'))
const CollectionList = lazy(() => import('@/pages/CollectionList'))
const CollectionView = lazy(() => import('@/pages/CollectionView'))
const CollectionNew = lazy(() => import('@/pages/CollectionNew'))
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Bookmarks = lazy(() => import('@/pages/Bookmarks'))
const Subscriptions = lazy(() => import('@/pages/Subscriptions'))
const Onboarding = lazy(() => import('@/pages/Onboarding'))
const Profile = lazy(() => import('@/pages/Profile'))
const ProfilePrivacy = lazy(() => import('@/pages/ProfilePrivacy'))
const OAuthAuthorize = lazy(() => import('@/pages/OAuthAuthorize'))
const ConnectedApps = lazy(() => import('@/pages/ConnectedApps'))
const MyApps = lazy(() => import('@/pages/MyApps'))
const AdminApps = lazy(() => import('@/pages/AdminApps'))
const NotFound = lazy(() => import('@/pages/NotFound'))

function LoggedIn() {
  return (
    <AuthGuard>
      <Outlet />
    </AuthGuard>
  )
}

export default function App() {
  return (
    <>
      <div className="min-h-screen flex flex-col">
        <TrialBanner />
        <Navigation />
        <main className="flex-1">
          <Suspense fallback={<p className="p-8 text-center text-gray-700">Loading…</p>}>
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/about" element={<About />} />
              <Route path="/accessibility" element={<Accessibility />} />
              <Route path="/api-docs" element={<ApiDocs />} />
              <Route path="/attributions" element={<Attributions />} />
              <Route path="/contact" element={<Contact />} />
              <Route path="/feedback" element={<Feedback />} />
              <Route path="/help" element={<Help />} />
              <Route path="/privacy" element={<Privacy />} />
              <Route path="/terms" element={<Terms />} />
              <Route path="/auth/login" element={<Login />} />
              <Route path="/auth/register" element={<Register />} />
              <Route path="/auth/activate" element={<Activate />} />
              <Route path="/auth/forgot-password" element={<ForgotPassword />} />
              <Route path="/auth/reset-password" element={<ResetPassword />} />
              {/* Handles its own login redirect, so it can come back to the same request */}
              <Route path="/oauth/authorize" element={<OAuthAuthorize />} />
              <Route path="/search" element={<Search />} />
              <Route path="/works" element={<WorkList />} />
              <Route path="/works/:id" element={<WorkView />} />
              <Route path="/series" element={<SeriesList />} />
              <Route path="/series/:id" element={<SeriesView />} />
              <Route path="/collections" element={<CollectionList />} />
              <Route path="/collections/:id" element={<CollectionView />} />

              <Route element={<LoggedIn />}>
                <Route path="/works/new" element={<WorkNew />} />
                <Route path="/works/:id/edit" element={<WorkEdit />} />
                <Route path="/series/new" element={<SeriesNew />} />
                <Route path="/series/:id/edit" element={<SeriesEdit />} />
                <Route path="/collections/new" element={<CollectionNew />} />
                <Route path="/dashboard" element={<Dashboard />} />
                <Route path="/bookmarks" element={<Bookmarks />} />
                <Route path="/subscriptions" element={<Subscriptions />} />
                <Route path="/onboarding" element={<Onboarding />} />
                <Route path="/profile" element={<Profile />} />
                <Route path="/profile/password" element={<ChangePassword />} />
                <Route path="/profile/privacy" element={<ProfilePrivacy />} />
                <Route path="/profile/apps" element={<ConnectedApps />} />
                <Route path="/profile/my-apps" element={<MyApps />} />
                <Route path="/admin/apps" element={<AdminApps />} />
              </Route>

              <Route path="*" element={<NotFound />} />
            </Routes>
          </Suspense>
        </main>
        <Footer />
      </div>
      <GDPRConsentBanner />
      <ServiceStatus />
      <PWAInit />
    </>
  )
}
