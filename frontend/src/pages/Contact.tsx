import SupportForm from '@/components/SupportForm';

export default function ContactPage() {
  return (
    <div className="max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">Contact</h1>
      <p className="mb-6 text-sm text-gray-700">
        Write to the people who run this archive. Choose &ldquo;Support and feedback&rdquo; for questions, problems with the
        site and ideas, or &ldquo;Policy and abuse&rdquo; to report something that breaks the rules. Only the people who
        answer messages can read what you send.
      </p>
      <SupportForm defaultKind="support" />
    </div>
  );
}
