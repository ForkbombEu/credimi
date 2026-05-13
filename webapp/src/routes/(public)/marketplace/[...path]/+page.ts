import { redirect } from '@/i18n';

export const load = async ({ params }) => {
  redirect(`/hub/${params.path}`);
};
