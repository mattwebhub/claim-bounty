import { ArrowRight, Boxes, Braces, ShieldCheck } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button, Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui';

const foundations = [
  { key: 'architecture', icon: Braces },
  { key: 'state', icon: Boxes },
  { key: 'quality', icon: ShieldCheck },
] as const;

export function Component() {
  const { t } = useTranslation('common');

  return (
    <main id="main-content" className="home-page" tabIndex={-1}>
      <section className="hero" aria-labelledby="hero-title">
        <p className="eyebrow">{t('home.eyebrow')}</p>
        <h1 id="hero-title">{t('home.title')}</h1>
        <p className="hero-copy">{t('home.description')}</p>
        <Button asChild size="lg">
          <a href="#foundations">
            {t('home.explore')}
            <ArrowRight aria-hidden="true" />
          </a>
        </Button>
      </section>

      <section id="foundations" className="foundation-grid" aria-labelledby="foundations-title">
        <h2 id="foundations-title" className="sr-only">
          {t('home.foundationsTitle')}
        </h2>
        {foundations.map(({ key, icon: Icon }) => (
          <Card key={key}>
            <CardHeader>
              <Icon className="feature-icon" aria-hidden="true" />
              <CardTitle>{t(`home.foundations.${key}.title`)}</CardTitle>
              <CardDescription>{t(`home.foundations.${key}.description`)}</CardDescription>
            </CardHeader>
            <CardContent>
              <code>{t(`home.foundations.${key}.command`)}</code>
            </CardContent>
          </Card>
        ))}
      </section>
    </main>
  );
}
