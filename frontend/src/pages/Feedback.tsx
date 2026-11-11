import SupportForm from '@/components/SupportForm';

export default function FeedbackPage() {
  return (
    <div className="max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">Feedback</h1>
      <p className="mb-6 text-sm text-gray-700">
        Tell us what works, what doesn&apos;t and what you would like to see. To report something that breaks the rules,
        use the <a href="/contact" className="underline text-orange-800">contact form</a> instead.
      </p>
      <SupportForm defaultKind="support" allowAbuse={false} />
    </div>
  );
}
