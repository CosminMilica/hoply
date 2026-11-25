<template>
  <div class="mx-auto flex w-full max-w-[620px] flex-col gap-4">
    <div class="flex flex-col gap-1.5">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Finding jobs is coming soon</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">
        We will search job boards for you, based on your profile: the roles, places and skills you care about.
      </p>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">
        For now, add any job you found yourself. We read it, match it to your profile, and you can tailor a CV for it.
      </p>
    </div>
    <RouterLink
      :to="{ name: 'hoply-job-add' }"
      class="flex w-full items-center gap-3.5 rounded-[14px] bg-hoply-600 px-4 py-3.5 text-left text-white hover:bg-hoply-700"
    >
      <span class="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-white/15">
        <UIcon name="i-lucide-link" class="size-5" />
      </span>
      <span class="flex min-w-0 flex-1 flex-col gap-0.5">
        <span class="text-[15px] font-extrabold">Add a job from a link or its text</span>
        <span class="text-[13px] leading-snug font-medium text-hoply-100">Seen it on LinkedIn, eJobs or a company site? Paste it here.</span>
      </span>
      <UIcon name="i-lucide-chevron-right" class="size-[18px] shrink-0" />
    </RouterLink>
  </div>
</template>
