import { createClient, ServerError, TransportError } from '@tygor/client';
import { registry } from './src/rpc/manifest';

// [snippet:client-setup]

const client = createClient(registry, {
  baseUrl: 'http://localhost:8080',
});

// [/snippet:client-setup]

// [snippet:client-calls]

async function demonstrateGenerics() {
  // PagedResponse[User] - Generic type instantiated for users
  const usersPage = await client.Users.List({
    page: 1,
    page_size: 10,
    role: 'admin'
  });

  console.log(`Users page ${usersPage.page}/${Math.ceil(usersPage.total / usersPage.page_size)}`);
  if (usersPage.data) {
    console.log(`Found ${usersPage.data.length} of ${usersPage.total} total`);
    usersPage.data.forEach(user => {
      console.log(`- ${user.username} (${user.email}) - ${user.role}`);
    });
  }

  // Result[User] - Generic type for success/error results
  const userResult = await client.Users.Get({ id: 1 });
  if (userResult.success && userResult.data) {
    console.log('User found:', userResult.data.username);
  } else if (userResult.error) {
    console.log('Error:', userResult.error);
  }

  // Result[Post] - Same generic type with different type parameter
  const postResult = await client.Posts.Create({
    title: 'My First Post',
    content: 'This demonstrates generic type instantiation',
    author_id: 1
  });

  if (postResult.success && postResult.data) {
    console.log('Post created:', postResult.data.title);
  }
}

// [/snippet:client-calls]

async function main() {
  try {
    await demonstrateGenerics();
  } catch (e: any) {
    if (e instanceof ServerError) {
      console.error('Server error:', e.code, e.message);
    } else if (e instanceof TransportError) {
      console.error('Transport error:', e.httpStatus);
    } else {
      console.error('Error:', e.message);
    }
  }
}

main();
